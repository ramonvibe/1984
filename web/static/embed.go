package static

import "embed"

//go:embed app.css app.js vendor/*
var Files embed.FS
