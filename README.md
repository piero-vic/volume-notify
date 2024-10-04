# `volume-notify`

A lightweight volume notifier daemon for PulseAudio. Intended to be used with window managers like i3 or Sway.

It requires a notification daemon like `mako` to be already installed in your system.

## Installation

```bash
go install .
```

## Usage

Add this line to your sway configuration to start `volume-notify` when starting Sway.

```bash
exec volume-notify
```
