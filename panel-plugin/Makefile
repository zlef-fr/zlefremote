# ZlefRemote xfce4-panel plugin — build & install.
#
#   make            # build the panel plugin (libzlefremote.so)
#   make standalone # build the standalone UI test harness (zr-standalone)
#   make install    # install plugin + .desktop + icon (honours DESTDIR/PREFIX)
#   make uninstall
#   make clean
#
# Build deps (Debian/Ubuntu):
#   sudo apt-get install build-essential pkg-config \
#        libgtk-3-dev libxfce4panel-2.0-dev libxfce4util-dev

PREFIX  ?= /usr
DESTDIR ?=
LIBDIR  ?= $(PREFIX)/lib
DATADIR ?= $(PREFIX)/share

PLUGIN_DIR  := $(LIBDIR)/xfce4/panel/plugins
DESKTOP_DIR := $(DATADIR)/xfce4/panel/plugins
ICON_DIR    := $(DATADIR)/icons/hicolor/scalable/apps

GTK_PKGS   := gtk+-3.0
PANEL_PKGS := libxfce4panel-2.0 gtk+-3.0

CC      ?= cc
CFLAGS  ?= -O2 -Wall -Wextra
CFLAGS  += -fPIC $(shell pkg-config --cflags $(PANEL_PKGS))
LDLIBS  := $(shell pkg-config --libs $(PANEL_PKGS))

GTK_CFLAGS := $(shell pkg-config --cflags $(GTK_PKGS))
GTK_LIBS   := $(shell pkg-config --libs $(GTK_PKGS))

PLUGIN_SO  := libzlefremote.so
UI_SRC     := zlefremote-ui.c
PLUGIN_SRC := zlefremote-plugin.c $(UI_SRC)

.PHONY: all standalone install uninstall clean

all: $(PLUGIN_SO)

$(PLUGIN_SO): $(PLUGIN_SRC) zlefremote.h
	$(CC) $(CFLAGS) -shared -o $@ $(PLUGIN_SRC) $(LDLIBS)

standalone: zr-standalone

zr-standalone: zlefremote-standalone.c $(UI_SRC) zlefremote.h
	$(CC) -O2 -Wall $(GTK_CFLAGS) -o $@ zlefremote-standalone.c $(UI_SRC) $(GTK_LIBS)

install: $(PLUGIN_SO)
	install -d $(DESTDIR)$(PLUGIN_DIR)
	install -m 0755 $(PLUGIN_SO) $(DESTDIR)$(PLUGIN_DIR)/$(PLUGIN_SO)
	install -d $(DESTDIR)$(DESKTOP_DIR)
	install -m 0644 zlefremote.desktop.in $(DESTDIR)$(DESKTOP_DIR)/zlefremote.desktop
	# install the whole hicolor tree (PNG sizes + scalable SVG). PNGs make the
	# panel icon render without an external SVG loader (librsvg/glycin).
	cd icons/hicolor && find . -type f | while read -r f; do \
	  install -Dm644 "$$f" "$(DESTDIR)$(DATADIR)/icons/hicolor/$$f"; done
	-gtk-update-icon-cache -f -t $(DESTDIR)$(DATADIR)/icons/hicolor 2>/dev/null || true

uninstall:
	rm -f $(DESTDIR)$(PLUGIN_DIR)/$(PLUGIN_SO)
	rm -f $(DESTDIR)$(DESKTOP_DIR)/zlefremote.desktop
	rm -f $(DESTDIR)$(DATADIR)/icons/hicolor/*/apps/zlefremote.png
	rm -f $(DESTDIR)$(ICON_DIR)/zlefremote.svg

clean:
	rm -f $(PLUGIN_SO) zr-standalone
