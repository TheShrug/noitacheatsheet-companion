package noita

import (
	"os"
	"strings"
	"testing"
)

// Expected values are the table settled in the issue comment before this
// ticket started, read off internal/noita/testdata/player.xml.
func TestParseWands(t *testing.T) {
	f, err := os.Open("testdata/player.xml")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	wands, err := ParseWands(f)
	if err != nil {
		t.Fatalf("ParseWands: %v", err)
	}

	if len(wands) != 4 {
		t.Fatalf("got %d wands, want 4", len(wands))
	}

	wantPerks := []string{"GLASS_CANNON", "EXTRA_HP"}

	cases := []struct {
		name          string
		capacity      int
		spellsPerCast int
		shuffle       bool
		rechargeTime  float64
		castDelay     float64
		spread        int
		maxMana       int
		sprite        string
		spellIDs      []string
	}{
		{
			name:          "handgun wand",
			capacity:      3,
			spellsPerCast: 1,
			shuffle:       false,
			rechargeTime:  0.37, // 22 / 60
			castDelay:     0.17, // 10 / 60
			spread:        0,
			maxMana:       117,
			sprite:        "data/items_gfx/handgun.png",
			spellIDs:      []string{"LIGHT_BULLET", "LIGHT_BULLET", "LIGHT_BULLET"},
		},
		{
			name:          "bomb wand",
			capacity:      1,
			spellsPerCast: 1,
			shuffle:       true,
			rechargeTime:  0.02, // 1 / 60
			castDelay:     0.12, // 7 / 60
			spread:        0,
			maxMana:       110,
			sprite:        "data/items_gfx/bomb_wand.png",
			spellIDs:      []string{"BOMB"},
		},
		{
			name:          "big wand",
			capacity:      9,
			spellsPerCast: 4,
			shuffle:       false,
			rechargeTime:  2.55, // 153 / 60
			castDelay:     0.18, // 11 / 60
			spread:        0,
			maxMana:       330,
			sprite:        "data/items_gfx/wands/wand_0968.png",
			spellIDs:      []string{"BOUNCE_LIGHTNING", "GRENADE", "ORBIT_FIREBALLS", "BUCKSHOT", "GLITTER_BOMB", "FIREBOMB"},
		},
		{
			name:          "slimeball wand",
			capacity:      6,
			spellsPerCast: 1,
			shuffle:       true,
			rechargeTime:  0.82, // 49 / 60
			castDelay:     0.05, // 3 / 60
			spread:        3,
			maxMana:       160,
			sprite:        "data/items_gfx/wands/wand_0724.png",
			spellIDs:      []string{"SLIMEBALL", "SLIMEBALL"},
		},
	}

	for i, want := range cases {
		got := wands[i]
		t.Run(want.name, func(t *testing.T) {
			if got.Capacity != want.capacity {
				t.Errorf("Capacity = %d, want %d", got.Capacity, want.capacity)
			}
			if got.SpellsPerCast != want.spellsPerCast {
				t.Errorf("SpellsPerCast = %d, want %d", got.SpellsPerCast, want.spellsPerCast)
			}
			if got.Shuffle != want.shuffle {
				t.Errorf("Shuffle = %v, want %v", got.Shuffle, want.shuffle)
			}
			if got.RechargeTime != want.rechargeTime {
				t.Errorf("RechargeTime = %v, want %v", got.RechargeTime, want.rechargeTime)
			}
			if got.CastDelay != want.castDelay {
				t.Errorf("CastDelay = %v, want %v", got.CastDelay, want.castDelay)
			}
			if got.Spread != want.spread {
				t.Errorf("Spread = %d, want %d", got.Spread, want.spread)
			}
			if got.MaxMana != want.maxMana {
				t.Errorf("MaxMana = %d, want %d", got.MaxMana, want.maxMana)
			}
			if got.SpritePath != want.sprite {
				t.Errorf("SpritePath = %q, want %q", got.SpritePath, want.sprite)
			}

			if len(got.Spells) != len(want.spellIDs) {
				t.Fatalf("got %d spells, want %d", len(got.Spells), len(want.spellIDs))
			}
			for j, spell := range got.Spells {
				if spell.ActionID != want.spellIDs[j] {
					t.Errorf("Spells[%d].ActionID = %q, want %q", j, spell.ActionID, want.spellIDs[j])
				}
				if spell.Slot != j {
					t.Errorf("Spells[%d].Slot = %d, want %d", j, spell.Slot, j)
				}
				if spell.AlwaysCast {
					t.Errorf("Spells[%d].AlwaysCast = true, want false", j)
				}
			}

			if len(got.Perks) != len(wantPerks) {
				t.Fatalf("got %d perks, want %d", len(got.Perks), len(wantPerks))
			}
			for j, perk := range got.Perks {
				if perk != wantPerks[j] {
					t.Errorf("Perks[%d] = %q, want %q", j, perk, wantPerks[j])
				}
			}
		})
	}
}

func TestParseWandsInvalidXML(t *testing.T) {
	_, err := ParseWands(strings.NewReader("not xml"))
	if err == nil {
		t.Fatal("expected an error for malformed XML, got nil")
	}
}
