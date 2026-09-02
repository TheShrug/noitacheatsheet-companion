# The toolchain lives here, not on anyone's laptop.
#
# Three stages, matching the fleet's container shape: a shared `base` so dev and
# release cannot drift, a `dev` stage you work inside, and a `build` stage that
# produces the binaries a player downloads.
#
# The Go version is pinned in exactly one place — go.mod — and read from there
# by CI (`go-version-file`). Bump it there, then here.

# --- base -------------------------------------------------------------------
# Shared toolchain. Anything both dev and release need goes here, so a release
# can never be built by a different compiler than the one tests ran under.
FROM golang:1.24-bookworm AS base

WORKDIR /src

# mingw-w64 is the Windows cross-compiler. It is not needed today: with no cgo
# dependency, `GOOS=windows go build` is pure Go and needs no C toolchain at
# all. It is installed anyway because the tray ticket adds one, and at that
# point Windows builds need a C compiler or they stop working — with an error
# about gcc, a long way from the change that caused it.
RUN apt-get update \
 && apt-get install --no-install-recommends -y gcc-mingw-w64-x86-64 \
 && rm -rf /var/lib/apt/lists/*

# Dependencies before source, so an edit to a .go file does not re-download the
# module cache. There are no dependencies yet; this layer is here so the first
# one added does not also rewrite the Dockerfile.
COPY go.mod go.sum* ./
RUN go mod download

# --- dev --------------------------------------------------------------------
# The environment `make` is meant to run in. Source is bind-mounted over /src
# by the devcontainer, so nothing is COPYed in.
FROM base AS dev

# make is the repo's interface (see CLAUDE.md); gh backs the /ticket skill and
# every workflow that touches the queue.
RUN apt-get update \
 && apt-get install --no-install-recommends -y make git ca-certificates gh \
 && rm -rf /var/lib/apt/lists/*

CMD ["sleep", "infinity"]

# --- build ------------------------------------------------------------------
# Produces the release binaries. Nothing here is interactive and nothing here
# is bind-mounted: what goes in is what is committed.
FROM base AS build

ARG VERSION=dev

COPY . .

# -trimpath strips the builder's absolute paths out of the binary, so two
# machines building the same commit produce the same bytes. -s -w drop the
# symbol table and DWARF, which is most of the size.
#
# CGO_ENABLED=0 gives a genuinely static binary that runs on any glibc or musl.
# The tray ticket has to turn this on for Linux and Windows and take the
# portability hit; that is a decision for its ADR, not a default to inherit.
RUN set -eux; \
    mkdir -p /out; \
    for target in linux/amd64 windows/amd64; do \
      os="${target%/*}"; arch="${target#*/}"; \
      ext=""; [ "$os" = "windows" ] && ext=".exe"; \
      CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        go build -trimpath \
          -ldflags "-s -w -X main.version=${VERSION}" \
          -o "/out/companion-${os}-${arch}${ext}" \
          ./cmd/companion; \
    done

# --- dist -------------------------------------------------------------------
# Nothing but the binaries, so `docker build --target dist --output dist .`
# writes them straight onto the host.
FROM scratch AS dist
COPY --from=build /out/ /
