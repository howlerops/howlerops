package main

import "embed"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/howlerops-light.png assets/howlerops-dark.png assets/howlerops-transparent.png
var iconFS embed.FS
