package world

import (
	"math"
	"math/rand"

	perlin "github.com/aquilax/go-perlin"
)

// WorldGenerator generates terrain for chunks
type WorldGenerator struct {
	Seed         int64
	heightNoise  *perlin.Perlin
	caveNoise    *perlin.Perlin
	biomeNoise   *perlin.Perlin
	detailNoise  *perlin.Perlin
	treeRand     *rand.Rand
	SeaLevel     int
	BaseHeight   int
	HeightScale  float64
}

// NewWorldGenerator creates a new terrain generator
func NewWorldGenerator(seed int64) *WorldGenerator {
	return &WorldGenerator{
		Seed:        seed,
		heightNoise: perlin.NewPerlin(2, 2, 3, seed),
		caveNoise:   perlin.NewPerlin(2, 2, 3, seed+1),
		biomeNoise:  perlin.NewPerlin(2, 2, 2, seed+2),
		detailNoise: perlin.NewPerlin(2, 2, 4, seed+3),
		treeRand:    rand.New(rand.NewSource(seed)),
		SeaLevel:    64,
		BaseHeight:  64,
		HeightScale: 40,
	}
}

// GenerateChunk generates terrain for a chunk
func (g *WorldGenerator) GenerateChunk(chunk *Chunk) {
	heightMap := make([][]int, ChunkSize)
	biomeMap := make([][]float64, ChunkSize)

	// Generate height and biome maps
	for x := 0; x < ChunkSize; x++ {
		heightMap[x] = make([]int, ChunkSize)
		biomeMap[x] = make([]float64, ChunkSize)

		for z := 0; z < ChunkSize; z++ {
			worldX := float64(chunk.X*ChunkSize + x)
			worldZ := float64(chunk.Z*ChunkSize + z)

			// Multi-octave noise for height
			height := g.getHeight(worldX, worldZ)
			heightMap[x][z] = height

			// Biome noise
			biomeMap[x][z] = g.biomeNoise.Noise2D(worldX*0.005, worldZ*0.005)
		}
	}

	// Generate blocks
	for x := 0; x < ChunkSize; x++ {
		for z := 0; z < ChunkSize; z++ {
			height := heightMap[x][z]
			biome := biomeMap[x][z]
			worldX := float64(chunk.X*ChunkSize + x)
			worldZ := float64(chunk.Z*ChunkSize + z)

			for y := 0; y < ChunkHeight; y++ {
				block := g.getBlock(x, y, z, height, biome, worldX, worldZ)
				chunk.Blocks[x][y][z] = block
			}
		}
	}

	// Generate caves
	g.generateCaves(chunk)

	// Generate ores
	g.generateOres(chunk)

	// Generate trees
	g.generateTrees(chunk, heightMap, biomeMap)

	chunk.MeshDirty = true
}

// getHeight calculates terrain height at a world position
func (g *WorldGenerator) getHeight(x, z float64) int {
	// Large-scale terrain
	continentNoise := g.heightNoise.Noise2D(x*0.002, z*0.002)

	// Medium-scale features
	hillNoise := g.heightNoise.Noise2D(x*0.01, z*0.01) * 0.5

	// Small-scale detail
	detailNoise := g.detailNoise.Noise2D(x*0.05, z*0.05) * 0.2

	// Combine noise
	noise := continentNoise + hillNoise + detailNoise

	// Calculate final height
	height := g.BaseHeight + int(noise*g.HeightScale)

	// Clamp height
	if height < 1 {
		height = 1
	}
	if height > ChunkHeight-10 {
		height = ChunkHeight - 10
	}

	return height
}

// getBlock determines the block type at a position
func (g *WorldGenerator) getBlock(x, y, z, height int, biome float64, worldX, worldZ float64) BlockType {
	// Bedrock layer
	if y == 0 {
		return BlockBedrock
	}
	if y < 5 {
		// Random bedrock in bottom layers
		if g.detailNoise.Noise3D(worldX*0.5, float64(y)*0.5, worldZ*0.5) > 0.3 {
			return BlockBedrock
		}
	}

	// Water
	if y < g.SeaLevel && y > height {
		return BlockWater
	}

	// Air above terrain
	if y > height {
		return BlockAir
	}

	// Determine biome type for surface blocks
	biomeType := g.GetBiome(worldX, worldZ)

	// Surface layer
	if y == height {
		switch biomeType {
		case BiomeOcean:
			return BlockSand
		case BiomeDesert:
			return BlockSand
		case BiomeSnow:
			return BlockSnow
		case BiomeMountain:
			if y > g.BaseHeight+35 {
				return BlockSnow
			}
			return BlockStone
		case BiomeSwamp:
			return BlockClay
		case BiomeJungle:
			return BlockGrass
		default:
			if y < g.SeaLevel+2 {
				return BlockSand
			}
			return BlockGrass
		}
	}

	// Sub-surface layers
	if y > height-4 {
		switch biomeType {
		case BiomeDesert, BiomeOcean:
			return BlockSand
		case BiomeMountain:
			if y > height-2 {
				return BlockGravel
			}
			return BlockStone
		case BiomeSwamp:
			if y > height-2 {
				return BlockClay
			}
			return BlockDirt
		case BiomeJungle:
			return BlockDirt
		default:
			if y < g.SeaLevel+2 {
				return BlockSand
			}
			return BlockDirt
		}
	}

	// Deep stone
	return BlockStone
}

// generateCaves generates cave systems
func (g *WorldGenerator) generateCaves(chunk *Chunk) {
	for x := 0; x < ChunkSize; x++ {
		for z := 0; z < ChunkSize; z++ {
			worldX := float64(chunk.X*ChunkSize + x)
			worldZ := float64(chunk.Z*ChunkSize + z)

			for y := 1; y < 60; y++ { // Caves below y=60
				if chunk.Blocks[x][y][z] == BlockAir || chunk.Blocks[x][y][z] == BlockWater {
					continue
				}

				// 3D noise for caves
				caveValue := g.caveNoise.Noise3D(worldX*0.05, float64(y)*0.05, worldZ*0.05)
				caveValue += g.caveNoise.Noise3D(worldX*0.1, float64(y)*0.1, worldZ*0.1) * 0.5

				// Carve cave if noise is high enough
				if caveValue > 0.6 {
					chunk.Blocks[x][y][z] = BlockAir
				}
			}
		}
	}
}

// generateOres places ore blocks
func (g *WorldGenerator) generateOres(chunk *Chunk) {
	ores := []struct {
		block    BlockType
		minY     int
		maxY     int
		rarity   float64
		veinSize int
	}{
		{BlockCoal, 5, 128, 0.02, 8},
		{BlockIron, 5, 64, 0.015, 6},
		{BlockGold, 5, 32, 0.005, 4},
		{BlockDiamond, 5, 16, 0.002, 3},
	}

	for _, ore := range ores {
		g.placeOreVeins(chunk, ore.block, ore.minY, ore.maxY, ore.rarity, ore.veinSize)
	}
}

// placeOreVeins places veins of ore in the chunk
func (g *WorldGenerator) placeOreVeins(chunk *Chunk, block BlockType, minY, maxY int, rarity float64, veinSize int) {
	rng := rand.New(rand.NewSource(g.Seed + int64(chunk.X*1000+chunk.Z) + int64(block)*100))

	attempts := int(float64(ChunkSize*ChunkSize*(maxY-minY)) * rarity)

	for i := 0; i < attempts; i++ {
		x := rng.Intn(ChunkSize)
		y := minY + rng.Intn(maxY-minY)
		z := rng.Intn(ChunkSize)

		// Place vein
		for j := 0; j < veinSize; j++ {
			vx := x + rng.Intn(3) - 1
			vy := y + rng.Intn(3) - 1
			vz := z + rng.Intn(3) - 1

			if vx >= 0 && vx < ChunkSize && vy >= minY && vy < maxY && vz >= 0 && vz < ChunkSize {
				if chunk.Blocks[vx][vy][vz] == BlockStone {
					chunk.Blocks[vx][vy][vz] = block
				}
			}

			x = vx
			y = vy
			z = vz
		}
	}
}

// generateTrees places trees on the terrain
func (g *WorldGenerator) generateTrees(chunk *Chunk, heightMap [][]int, biomeMap [][]float64) {
	rng := rand.New(rand.NewSource(g.Seed + int64(chunk.X*10000+chunk.Z)))

	for x := 2; x < ChunkSize-2; x++ {
		for z := 2; z < ChunkSize-2; z++ {
			height := heightMap[x][z]

			// Don't place trees underwater
			if height < g.SeaLevel {
				continue
			}

			worldX := float64(chunk.X*ChunkSize + x)
			worldZ := float64(chunk.Z*ChunkSize + z)
			biomeType := g.GetBiome(worldX, worldZ)

			var treeChance float64
			switch biomeType {
			case BiomeForest:
				treeChance = 0.06 // Dense forest
			case BiomePlains:
				treeChance = 0.015 // Sparse trees
			case BiomeDesert:
				treeChance = 0.005 // Rare cactus-like
			case BiomeSwamp:
				treeChance = 0.03 // Moderate swamp trees
			case BiomeJungle:
				treeChance = 0.09 // Very dense jungle
			default:
				continue // No trees in snow, mountain, ocean
			}

			if rng.Float64() < treeChance {
				switch biomeType {
				case BiomeDesert:
					g.placeCactus(chunk, x, height+1, z, rng)
				case BiomeSwamp:
					g.placeSwampTree(chunk, x, height+1, z, rng)
				case BiomeJungle:
					g.placeJungleTree(chunk, x, height+1, z, rng)
				default:
					g.placeTree(chunk, x, height+1, z, rng)
				}
			}
		}
	}
}

// placeTree places a tree at the given position
func (g *WorldGenerator) placeTree(chunk *Chunk, x, y, z int, rng *rand.Rand) {
	trunkHeight := 4 + rng.Intn(3)

	// Check if there's room for the tree
	if y+trunkHeight+3 >= ChunkHeight {
		return
	}

	// Place trunk
	for dy := 0; dy < trunkHeight; dy++ {
		if y+dy < ChunkHeight {
			chunk.Blocks[x][y+dy][z] = BlockWood
		}
	}

	// Place leaves
	leafStart := trunkHeight - 2
	for dy := leafStart; dy < trunkHeight+2; dy++ {
		radius := 2
		if dy >= trunkHeight {
			radius = 1
		}

		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				// Skip corners for rounder shape
				if dx*dx+dz*dz > radius*radius+1 {
					continue
				}

				lx, ly, lz := x+dx, y+dy, z+dz
				if lx >= 0 && lx < ChunkSize && ly >= 0 && ly < ChunkHeight && lz >= 0 && lz < ChunkSize {
					if chunk.Blocks[lx][ly][lz] == BlockAir {
						chunk.Blocks[lx][ly][lz] = BlockLeaves
					}
				}
			}
		}
	}
}

// placeCactus places a cactus (sand pillar) at the given position
func (g *WorldGenerator) placeCactus(chunk *Chunk, x, y, z int, rng *rand.Rand) {
	height := 2 + rng.Intn(2) // 2-3 blocks tall
	if y+height >= ChunkHeight {
		return
	}
	for dy := 0; dy < height; dy++ {
		chunk.Blocks[x][y+dy][z] = BlockSand
	}
}

// placeSwampTree places a short, wide swamp tree
func (g *WorldGenerator) placeSwampTree(chunk *Chunk, x, y, z int, rng *rand.Rand) {
	trunkHeight := 3 + rng.Intn(2) // 3-4 blocks tall (short)

	if y+trunkHeight+2 >= ChunkHeight {
		return
	}

	// Place trunk
	for dy := 0; dy < trunkHeight; dy++ {
		if y+dy < ChunkHeight {
			chunk.Blocks[x][y+dy][z] = BlockWood
		}
	}

	// Wide, flat leaf canopy
	for dy := trunkHeight - 1; dy < trunkHeight+2; dy++ {
		radius := 3
		if dy >= trunkHeight {
			radius = 2
		}

		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx*dx+dz*dz > radius*radius+1 {
					continue
				}
				lx, ly, lz := x+dx, y+dy, z+dz
				if lx >= 0 && lx < ChunkSize && ly >= 0 && ly < ChunkHeight && lz >= 0 && lz < ChunkSize {
					if chunk.Blocks[lx][ly][lz] == BlockAir {
						chunk.Blocks[lx][ly][lz] = BlockLeaves
					}
				}
			}
		}
	}
}

// placeJungleTree places a tall jungle tree with dense foliage
func (g *WorldGenerator) placeJungleTree(chunk *Chunk, x, y, z int, rng *rand.Rand) {
	trunkHeight := 7 + rng.Intn(5) // 7-11 blocks tall

	if y+trunkHeight+3 >= ChunkHeight {
		return
	}

	// Place trunk
	for dy := 0; dy < trunkHeight; dy++ {
		if y+dy < ChunkHeight {
			chunk.Blocks[x][y+dy][z] = BlockWood
		}
	}

	// Dense leaf canopy at the top
	for dy := trunkHeight - 3; dy < trunkHeight+3; dy++ {
		radius := 3
		if dy < trunkHeight-1 {
			radius = 1
		} else if dy >= trunkHeight+1 {
			radius = 1
		}

		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx*dx+dz*dz > radius*radius+1 {
					continue
				}
				lx, ly, lz := x+dx, y+dy, z+dz
				if lx >= 0 && lx < ChunkSize && ly >= 0 && ly < ChunkHeight && lz >= 0 && lz < ChunkSize {
					if chunk.Blocks[lx][ly][lz] == BlockAir {
						chunk.Blocks[lx][ly][lz] = BlockLeaves
					}
				}
			}
		}
	}
}

// GenerateStructure generates special structures (future use)
func (g *WorldGenerator) GenerateStructure(chunk *Chunk, structureType string) {
	// Placeholder for future structure generation
	switch structureType {
	case "village":
		// Generate village
	case "dungeon":
		// Generate dungeon
	}
}

// GetSpawnPoint finds a suitable spawn point
func (g *WorldGenerator) GetSpawnPoint() (float64, float64, float64) {
	// Find a spot near origin that's above water
	for x := -10; x < 10; x++ {
		for z := -10; z < 10; z++ {
			height := g.getHeight(float64(x), float64(z))
			if height > g.SeaLevel+2 {
				return float64(x) + 0.5, float64(height) + 2, float64(z) + 0.5
			}
		}
	}
	return 0.5, float64(g.BaseHeight) + 10, 0.5
}

// Biome types
type BiomeType int

const (
	BiomePlains BiomeType = iota
	BiomeForest
	BiomeDesert
	BiomeSnow
	BiomeMountain
	BiomeOcean
	BiomeSwamp
	BiomeJungle
)

// GetBiome returns the biome at a world position
func (g *WorldGenerator) GetBiome(x, z float64) BiomeType {
	temp := g.biomeNoise.Noise2D(x*0.005, z*0.005)
	height := g.getHeight(x, z)

	if height < g.SeaLevel-5 {
		return BiomeOcean
	}
	if height > g.BaseHeight+30 {
		return BiomeMountain
	}
	if temp > 0.5 {
		return BiomeSnow
	}
	if temp < -0.3 {
		return BiomeDesert
	}
	// Swamp: warm-ish low-lying areas near sea level
	if temp > 0.2 && temp <= 0.5 && height < g.SeaLevel+4 {
		return BiomeSwamp
	}
	// Jungle: hot areas (between desert and moderate) with decent elevation
	if temp < -0.1 && temp >= -0.3 && height > g.SeaLevel {
		return BiomeJungle
	}
	if math.Abs(temp) < 0.2 {
		return BiomeForest
	}
	return BiomePlains
}
