// PARITY: this file is a port of WandService.ParseSaveFile and its helpers
// GetSpellsFromInventoryItem and GetAllPlayerPerks, in
// NoitaCheatSheet/Client/Services/WandService.cs in TheShrug/NoitaSpellCasters.
// See .claude/skills/parity/SKILL.md and ADR 1.
//
// The site resolves spell and perk identifiers against its own enum, and that
// enum is what this port deliberately does not carry: every identifier here
// (e.g. "LIGHT_BULLET", "GLASS_CANNON") is left exactly as the game wrote it,
// for the server to resolve.
package noita

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// framesPerSecond is the fixed rate Noita's simulation runs at, independent
// of display refresh rate. player.xml stores delays in frames.
const framesPerSecond = 60

// Wand is one wand entry exactly as the game describes it.
type Wand struct {
	SpritePath      string
	Capacity        int
	SpellsPerCast   int
	Shuffle         bool
	MaxMana         int
	ManaChargeSpeed int
	CastDelay       float64
	RechargeTime    float64
	Spread          int
	Spells          []WandSpell
	Perks           []string
}

// WandSpell is one card in a wand's deck.
type WandSpell struct {
	ActionID   string
	Slot       int
	AlwaysCast bool
}

// node is a generic XML element: a name, its attributes, and its children.
// player.xml nests wands, their spells, and the player's perks inside the
// player Entity at varying depths, so a fixed struct per element (as
// encoding/xml normally wants) can't express "find the nearest
// AbilityComponent below here" — this can, by walking the tree the same way
// the C# XElement.Descendants() does.
type node struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []node     `xml:",any"`
}

func (n node) attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func (n node) attrInt(name string) int {
	v, _ := n.attr(name)
	i, _ := strconv.Atoi(v)
	return i
}

func (n node) attrFloat(name string) float64 {
	v, _ := n.attr(name)
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

// attrBool matches the site's BoolExtensionMethods.Parse: only "1", "true"
// and "True" are true — everything else, including a missing attribute, is
// false.
func (n node) attrBool(name string) bool {
	v, _ := n.attr(name)
	return v == "1" || v == "true" || v == "True"
}

// hasTag reports whether n's tags attribute contains name as a whole
// comma-separated token, not a substring — so a tag like
// "not_enabled_in_wand" cannot be mistaken for "wand".
func hasTag(n node, name string) bool {
	tags, _ := n.attr("tags")
	for _, t := range strings.Split(tags, ",") {
		if t == name {
			return true
		}
	}
	return false
}

// descendants returns every element named tag anywhere below n, in document
// order — matching XElement.Descendants().
func descendants(n node, tag string) []node {
	var out []node
	for _, c := range n.Children {
		if c.XMLName.Local == tag {
			out = append(out, c)
		}
		out = append(out, descendants(c, tag)...)
	}
	return out
}

// firstDescendant returns the first descendant named tag, or the zero node
// if there is none — attribute lookups on the zero node just come back
// empty, which is what lets the fields below skip nil checks.
func firstDescendant(n node, tag string) node {
	all := descendants(n, tag)
	if len(all) == 0 {
		return node{}
	}
	return all[0]
}

// ParseWands reads a player.xml and returns the wands it describes, in the
// order they appear in the save. It takes an io.Reader rather than a path so
// the gif watcher and the tests can both drive it without a real file.
func ParseWands(r io.Reader) ([]Wand, error) {
	var root node
	if err := xml.NewDecoder(r).Decode(&root); err != nil {
		return nil, fmt.Errorf("parsing player.xml: %w", err)
	}

	perks := allPerks(root)

	var wands []Wand
	for _, item := range descendants(root, "Entity") {
		if !hasTag(item, "wand") {
			continue
		}

		ability := firstDescendant(item, "AbilityComponent")
		gunConfig := firstDescendant(ability, "gun_config")
		gunAction := firstDescendant(ability, "gunaction_config")

		wands = append(wands, Wand{
			SpritePath:      spritePath(ability),
			Capacity:        gunConfig.attrInt("deck_capacity"),
			SpellsPerCast:   gunConfig.attrInt("actions_per_round"),
			Shuffle:         gunConfig.attrBool("shuffle_deck_when_empty"),
			MaxMana:         int(math.Floor(ability.attrFloat("mana_max"))),
			ManaChargeSpeed: ability.attrInt("mana_charge_speed"),
			CastDelay:       round2(gunAction.attrFloat("fire_rate_wait") / framesPerSecond),
			RechargeTime:    round2(gunConfig.attrFloat("reload_time") / framesPerSecond),
			Spread:          gunAction.attrInt("spread_degrees"),
			Spells:          spells(item),
			Perks:           perks,
		})
	}

	return wands, nil
}

// spritePath maps sprite_file onto the art it names. It's either the .png
// directly or a Noita <Sprite> .xml wrapping one, in which case the art is
// the same stem with the extension swapped — except the "<name>_sprite.xml"
// family, whose art drops that suffix too.
func spritePath(ability node) string {
	raw, ok := ability.attr("sprite_file")
	if !ok {
		return ""
	}
	stem := strings.SplitN(raw, ".", 2)[0]
	stem = strings.TrimSuffix(stem, "_sprite")
	return stem + ".png"
}

// spells reads one wand's deck: every descendant Entity tagged card_action.
func spells(wand node) []WandSpell {
	var out []WandSpell
	for _, card := range descendants(wand, "Entity") {
		if !hasTag(card, "card_action") {
			continue
		}

		action := firstDescendant(card, "ItemActionComponent")
		item := firstDescendant(card, "ItemComponent")
		actionID, _ := action.attr("action_id")

		out = append(out, WandSpell{
			ActionID:   actionID,
			Slot:       item.attrInt("inventory_slot.x"),
			AlwaysCast: item.attrBool("permanently_attached"),
		})
	}
	return out
}

// allPerks reads every perk the player has unlocked, anywhere in the save.
// Perks aren't per-wand — every wand in the result gets this same list,
// matching the site.
func allPerks(root node) []string {
	var out []string
	for _, icon := range descendants(root, "UIIconComponent") {
		if v, _ := icon.attr("is_perk"); v != "1" {
			continue
		}
		name, _ := icon.attr("name")
		out = append(out, strings.ToUpper(strings.ReplaceAll(name, "$perk_", "")))
	}
	return out
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
