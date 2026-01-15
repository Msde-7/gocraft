package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gocraft/pkg/engine"
)

func main() {
	// Parse command line flags
	width := flag.Int("width", 1280, "Window width")
	height := flag.Int("height", 720, "Window height")
	seed := flag.Int64("seed", 0, "World seed (0 for random)")
	flag.Parse()

	// Use random seed if not specified
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	fmt.Println("========================================")
	fmt.Println("     GoCraft - Minecraft in Go")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("World Seed: %d\n", *seed)
	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  WASD        - Move")
	fmt.Println("  Space       - Jump / Fly up")
	fmt.Println("  Shift       - Sneak / Fly down")
	fmt.Println("  Ctrl        - Sprint")
	fmt.Println("  F           - Toggle fly mode")
	fmt.Println("  Left Click  - Break block")
	fmt.Println("  Right Click - Place block")
	fmt.Println("  1-9         - Select hotbar slot")
	fmt.Println("  Scroll      - Change hotbar slot")
	fmt.Println("  Escape      - Release/capture mouse")
	fmt.Println("  Ctrl+Q      - Quit")
	fmt.Println()
	fmt.Println("Starting game...")
	fmt.Println()

	// Create and run game
	game, err := engine.NewGame(*width, *height, *seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start game: %v\n", err)
		os.Exit(1)
	}
	defer game.Cleanup()

	game.Run()

	fmt.Println("Thanks for playing!")
}
