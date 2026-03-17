package render

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Texture represents an OpenGL texture
type Texture struct {
	ID     uint32
	Width  int
	Height int
}

// NewTexture creates a texture from an image
func NewTexture(img image.Image) *Texture {
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	var texID uint32
	gl.GenTextures(1, &texID)
	gl.BindTexture(gl.TEXTURE_2D, texID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA,
		int32(rgba.Bounds().Dx()), int32(rgba.Bounds().Dy()),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

	gl.GenerateMipmap(gl.TEXTURE_2D)

	return &Texture{
		ID:     texID,
		Width:  rgba.Bounds().Dx(),
		Height: rgba.Bounds().Dy(),
	}
}

// Bind binds the texture to a texture unit
func (t *Texture) Bind(unit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit)
	gl.BindTexture(gl.TEXTURE_2D, t.ID)
}

// Delete releases the texture
func (t *Texture) Delete() {
	gl.DeleteTextures(1, &t.ID)
}

// GenerateBlockAtlas creates a texture atlas with all block textures
func GenerateBlockAtlas() *Texture {
	atlasSize := 16 // 16x16 textures
	texSize := 16   // Each texture is 16x16 pixels
	totalSize := atlasSize * texSize

	img := image.NewRGBA(image.Rect(0, 0, totalSize, totalSize))

	// Fill with transparent black (prevents magenta bleed through mipmaps)
	for y := 0; y < totalSize; y++ {
		for x := 0; x < totalSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	// Generate each texture
	textures := []struct {
		index int
		gen   func(x, y int) color.RGBA
	}{
		{0, genGrassTop},
		{1, genGrassSide},
		{2, genDirt},
		{3, genStone},
		{4, genCobblestone},
		{5, genWoodTop},
		{6, genWoodSide},
		{7, genLeaves},
		{8, genSand},
		{9, genWater},
		{10, genGlass},
		{11, genBrick},
		{12, genOakPlanks},
		{13, genSnowTop},
		{14, genSnowSide},
		{15, genCoalOre},
		{16, genIronOre},
		{17, genGoldOre},
		{18, genDiamondOre},
		{19, genBedrock},
		{20, genGravel},
		{21, genClay},
	}

	for _, tex := range textures {
		tx := (tex.index % atlasSize) * texSize
		ty := (tex.index / atlasSize) * texSize

		for y := 0; y < texSize; y++ {
			for x := 0; x < texSize; x++ {
				img.Set(tx+x, ty+y, tex.gen(x, y))
			}
		}
	}

	return NewTexture(img)
}

// Simple hash function for procedural generation
func hash(x, y, seed int) int {
	h := seed
	h ^= x * 374761393
	h ^= y * 668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h
}

func noise(x, y, seed int) float64 {
	return float64(hash(x, y, seed)&0xFFFF) / 65535.0
}

func genGrassTop(x, y int) color.RGBA {
	n := noise(x, y, 1)
	green := uint8(90 + n*40)
	return color.RGBA{50, green, 30, 255}
}

func genGrassSide(x, y int) color.RGBA {
	if y < 4 {
		// Grass top portion
		n := noise(x, y, 1)
		green := uint8(90 + n*40)
		return color.RGBA{50, green, 30, 255}
	}
	// Dirt portion
	return genDirt(x, y)
}

func genDirt(x, y int) color.RGBA {
	n := noise(x, y, 2)
	r := uint8(139 + n*20 - 10)
	g := uint8(90 + n*20 - 10)
	b := uint8(43 + n*10 - 5)
	return color.RGBA{r, g, b, 255}
}

func genStone(x, y int) color.RGBA {
	n := noise(x, y, 3)
	gray := uint8(120 + n*30 - 15)
	return color.RGBA{gray, gray, gray, 255}
}

func genCobblestone(x, y int) color.RGBA {
	n := noise(x, y, 4)
	n2 := noise(x/3, y/3, 5)
	gray := uint8(100 + n*40 + n2*20 - 30)
	return color.RGBA{gray, gray, gray, 255}
}

func genWoodTop(x, y int) color.RGBA {
	// Concentric rings
	cx, cy := 8, 8
	dx, dy := x-cx, y-cy
	dist := dx*dx + dy*dy
	ring := dist % 12
	if ring < 3 {
		return color.RGBA{139, 90, 43, 255}
	}
	return color.RGBA{180, 140, 80, 255}
}

func genWoodSide(x, y int) color.RGBA {
	n := noise(x, y, 6)
	if (y+int(n*4))%4 == 0 {
		return color.RGBA{100, 70, 30, 255}
	}
	return color.RGBA{139, 90, 43, 255}
}

func genLeaves(x, y int) color.RGBA {
	n := noise(x, y, 7)
	if n > 0.3 {
		green := uint8(60 + n*60)
		return color.RGBA{30, green, 20, 200}
	}
	return color.RGBA{40, 120, 30, 200}
}

func genSand(x, y int) color.RGBA {
	n := noise(x, y, 8)
	r := uint8(220 + n*20 - 10)
	g := uint8(200 + n*20 - 10)
	b := uint8(150 + n*20 - 10)
	return color.RGBA{r, g, b, 255}
}

func genWater(x, y int) color.RGBA {
	n := noise(x, y, 9)
	b := uint8(180 + n*40)
	return color.RGBA{30, 100, b, 180}
}

func genGlass(x, y int) color.RGBA {
	// Border
	if x == 0 || x == 15 || y == 0 || y == 15 {
		return color.RGBA{200, 220, 230, 255}
	}
	return color.RGBA{200, 220, 230, 100}
}

func genBrick(x, y int) color.RGBA {
	// Brick pattern
	brickH := 4
	brickW := 8
	mortarSize := 1

	yMod := y % brickH
	xOffset := 0
	if (y/brickH)%2 == 1 {
		xOffset = brickW / 2
	}
	xMod := (x + xOffset) % brickW

	// Mortar
	if yMod < mortarSize || xMod < mortarSize {
		return color.RGBA{180, 180, 180, 255}
	}

	// Brick with variation
	n := noise(x/brickW, y/brickH, 10)
	r := uint8(180 + n*40 - 20)
	g := uint8(80 + n*20 - 10)
	b := uint8(60 + n*20 - 10)
	return color.RGBA{r, g, b, 255}
}

func genOakPlanks(x, y int) color.RGBA {
	n := noise(x, y, 11)
	plankY := y % 4
	if plankY == 0 {
		return color.RGBA{150, 110, 50, 255}
	}
	r := uint8(180 + n*30 - 15)
	g := uint8(140 + n*20 - 10)
	b := uint8(70 + n*20 - 10)
	return color.RGBA{r, g, b, 255}
}

func genSnowTop(x, y int) color.RGBA {
	n := noise(x, y, 12)
	white := uint8(240 + n*15 - 8)
	return color.RGBA{white, white, white, 255}
}

func genSnowSide(x, y int) color.RGBA {
	if y < 4 {
		return genSnowTop(x, y)
	}
	return genDirt(x, y)
}

func genOre(x, y, seed int, baseColor, oreColor color.RGBA) color.RGBA {
	// Stone background with ore spots
	n := noise(x, y, seed)
	n2 := noise(x*2, y*2, seed+100)

	if n > 0.6 && n2 > 0.4 {
		return oreColor
	}
	return baseColor
}

func genCoalOre(x, y int) color.RGBA {
	return genOre(x, y, 13, genStone(x, y), color.RGBA{30, 30, 30, 255})
}

func genIronOre(x, y int) color.RGBA {
	return genOre(x, y, 14, genStone(x, y), color.RGBA{200, 180, 160, 255})
}

func genGoldOre(x, y int) color.RGBA {
	return genOre(x, y, 15, genStone(x, y), color.RGBA{255, 200, 50, 255})
}

func genDiamondOre(x, y int) color.RGBA {
	return genOre(x, y, 16, genStone(x, y), color.RGBA{100, 220, 255, 255})
}

func genBedrock(x, y int) color.RGBA {
	n := noise(x, y, 17)
	gray := uint8(40 + n*30)
	return color.RGBA{gray, gray, gray, 255}
}

func genGravel(x, y int) color.RGBA {
	n := noise(x, y, 18)
	gray := uint8(130 + n*40 - 20)
	return color.RGBA{gray, gray, gray, 255}
}

func genClay(x, y int) color.RGBA {
	n := noise(x, y, 19)
	r := uint8(160 + n*20 - 10)
	g := uint8(165 + n*20 - 10)
	b := uint8(180 + n*20 - 10)
	return color.RGBA{r, g, b, 255}
}
