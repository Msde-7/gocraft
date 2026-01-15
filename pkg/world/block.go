package world

// BlockType represents a type of block
type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
	BlockCobblestone
	BlockWood
	BlockLeaves
	BlockSand
	BlockWater
	BlockGlass
	BlockBrick
	BlockOakPlanks
	BlockSnow
	BlockCoal
	BlockIron
	BlockGold
	BlockDiamond
	BlockBedrock
	BlockGravel
	BlockClay
	BlockCount
)

// BlockInfo contains metadata about a block type
type BlockInfo struct {
	Name        string
	Solid       bool
	Transparent bool
	TexTop      int // Texture atlas index for top face
	TexBottom   int // Texture atlas index for bottom face
	TexSide     int // Texture atlas index for side faces
	LightLevel  uint8
}

// BlockInfos contains info for all block types
var BlockInfos = [BlockCount]BlockInfo{
	BlockAir:         {"Air", false, true, 0, 0, 0, 0},
	BlockGrass:       {"Grass", true, false, 0, 2, 1, 0},
	BlockDirt:        {"Dirt", true, false, 2, 2, 2, 0},
	BlockStone:       {"Stone", true, false, 3, 3, 3, 0},
	BlockCobblestone: {"Cobblestone", true, false, 4, 4, 4, 0},
	BlockWood:        {"Wood", true, false, 5, 5, 6, 0},
	BlockLeaves:      {"Leaves", true, true, 7, 7, 7, 0},
	BlockSand:        {"Sand", true, false, 8, 8, 8, 0},
	BlockWater:       {"Water", false, true, 9, 9, 9, 0},
	BlockGlass:       {"Glass", true, true, 10, 10, 10, 0},
	BlockBrick:       {"Brick", true, false, 11, 11, 11, 0},
	BlockOakPlanks:   {"Oak Planks", true, false, 12, 12, 12, 0},
	BlockSnow:        {"Snow", true, false, 13, 2, 14, 0},
	BlockCoal:        {"Coal Ore", true, false, 15, 15, 15, 0},
	BlockIron:        {"Iron Ore", true, false, 16, 16, 16, 0},
	BlockGold:        {"Gold Ore", true, false, 17, 17, 17, 0},
	BlockDiamond:     {"Diamond Ore", true, false, 18, 18, 18, 0},
	BlockBedrock:     {"Bedrock", true, false, 19, 19, 19, 0},
	BlockGravel:      {"Gravel", true, false, 20, 20, 20, 0},
	BlockClay:        {"Clay", true, false, 21, 21, 21, 0},
}

// IsSolid returns true if the block is solid
func (b BlockType) IsSolid() bool {
	if b >= BlockCount {
		return false
	}
	return BlockInfos[b].Solid
}

// IsTransparent returns true if the block is transparent
func (b BlockType) IsTransparent() bool {
	if b >= BlockCount {
		return true
	}
	return BlockInfos[b].Transparent
}

// GetName returns the block name
func (b BlockType) GetName() string {
	if b >= BlockCount {
		return "Unknown"
	}
	return BlockInfos[b].Name
}
