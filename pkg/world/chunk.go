package world

import (
	"sync"

	"gocraft/pkg/utils"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ChunkSize   = 16
	ChunkHeight = 256
)

// Chunk represents a 16x256x16 section of the world
type Chunk struct {
	X, Z   int
	Blocks [ChunkSize][ChunkHeight][ChunkSize]BlockType

	// Mesh data
	VAO, VBO, EBO uint32
	IndexCount    int32
	MeshDirty     bool
	MeshBuilt     bool

	// Threading
	mutex sync.RWMutex
}

// NewChunk creates a new chunk at the given position
func NewChunk(x, z int) *Chunk {
	return &Chunk{
		X:         x,
		Z:         z,
		MeshDirty: true,
	}
}

// GetBlock returns the block at local coordinates
func (c *Chunk) GetBlock(x, y, z int) BlockType {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkHeight || z < 0 || z >= ChunkSize {
		return BlockAir
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Blocks[x][y][z]
}

// SetBlock sets the block at local coordinates
func (c *Chunk) SetBlock(x, y, z int, block BlockType) {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkHeight || z < 0 || z >= ChunkSize {
		return
	}
	c.mutex.Lock()
	c.Blocks[x][y][z] = block
	c.MeshDirty = true
	c.mutex.Unlock()
}

// WorldPos returns the world position of the chunk
func (c *Chunk) WorldPos() mgl32.Vec3 {
	return mgl32.Vec3{float32(c.X * ChunkSize), 0, float32(c.Z * ChunkSize)}
}

// GetAABB returns the axis-aligned bounding box of the chunk
func (c *Chunk) GetAABB() utils.AABB {
	pos := c.WorldPos()
	return utils.AABB{
		Min: pos,
		Max: pos.Add(mgl32.Vec3{ChunkSize, ChunkHeight, ChunkSize}),
	}
}

// Vertex represents a vertex in the mesh
type Vertex struct {
	Position mgl32.Vec3
	TexCoord mgl32.Vec2
	Normal   mgl32.Vec3
	AO       float32
	Light    float32
}

// Face directions
const (
	FaceTop = iota
	FaceBottom
	FaceNorth
	FaceSouth
	FaceEast
	FaceWest
)

var faceNormals = []mgl32.Vec3{
	{0, 1, 0},  // Top
	{0, -1, 0}, // Bottom
	{0, 0, -1}, // North
	{0, 0, 1},  // South
	{1, 0, 0},  // East
	{-1, 0, 0}, // West
}

var faceVertices = [6][4]mgl32.Vec3{
	// Top face (Y+)
	{{0, 1, 0}, {1, 1, 0}, {1, 1, 1}, {0, 1, 1}},
	// Bottom face (Y-)
	{{0, 0, 1}, {1, 0, 1}, {1, 0, 0}, {0, 0, 0}},
	// North face (Z-)
	{{1, 0, 0}, {0, 0, 0}, {0, 1, 0}, {1, 1, 0}},
	// South face (Z+)
	{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}},
	// East face (X+)
	{{1, 0, 1}, {1, 0, 0}, {1, 1, 0}, {1, 1, 1}},
	// West face (X-)
	{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}},
}

// Texture atlas constants
const (
	AtlasSize    = 16 // 16x16 textures in atlas
	TextureSize  = 16 // Each texture is 16x16 pixels
	TexCoordSize = 1.0 / float32(AtlasSize)
)

// GetTexCoords returns texture coordinates for a texture index
func GetTexCoords(texIndex int) [4]mgl32.Vec2 {
	tx := float32(texIndex % AtlasSize)
	ty := float32(texIndex / AtlasSize)

	// Small offset to prevent texture bleeding
	offset := float32(0.001)

	return [4]mgl32.Vec2{
		{tx*TexCoordSize + offset, (ty+1)*TexCoordSize - offset},
		{(tx+1)*TexCoordSize - offset, (ty+1)*TexCoordSize - offset},
		{(tx+1)*TexCoordSize - offset, ty*TexCoordSize + offset},
		{tx*TexCoordSize + offset, ty*TexCoordSize + offset},
	}
}

// BuildMesh generates the mesh for the chunk
func (c *Chunk) BuildMesh(getNeighborBlock func(x, y, z int) BlockType) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Pre-allocate slices for better performance
	vertices := make([]float32, 0, 50000)
	indices := make([]uint32, 0, 30000)
	var indexOffset uint32 = 0

	// World-coordinate block lookup for cross-chunk boundaries and AO
	worldGetBlock := func(wx, wy, wz int) BlockType {
		// Check if within this chunk (fast path, no lock needed since we hold it)
		lx := wx - c.X*ChunkSize
		lz := wz - c.Z*ChunkSize
		if lx >= 0 && lx < ChunkSize && lz >= 0 && lz < ChunkSize && wy >= 0 && wy < ChunkHeight {
			return c.Blocks[lx][wy][lz]
		}
		// Cross-chunk: use the provided neighbor lookup
		return getNeighborBlock(wx, wy, wz)
	}

	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < 128; y++ { // Up to 128 height
			for z := 0; z < ChunkSize; z++ {
				block := c.Blocks[x][y][z]
				if block == BlockAir {
					continue
				}

				blockInfo := BlockInfos[block]
				worldX := c.X*ChunkSize + x
				worldZ := c.Z*ChunkSize + z

				// Check each face using world coordinates for cross-chunk accuracy
				for face := 0; face < 6; face++ {
					var neighbor BlockType

					switch face {
					case 0: // Top
						neighbor = worldGetBlock(worldX, y+1, worldZ)
					case 1: // Bottom
						if y > 0 {
							neighbor = c.Blocks[x][y-1][z]
						}
					case 2: // North (Z-)
						neighbor = worldGetBlock(worldX, y, worldZ-1)
					case 3: // South (Z+)
						neighbor = worldGetBlock(worldX, y, worldZ+1)
					case 4: // East (X+)
						neighbor = worldGetBlock(worldX+1, y, worldZ)
					case 5: // West (X-)
						neighbor = worldGetBlock(worldX-1, y, worldZ)
					}

					// Skip if neighbor is solid and opaque
					if neighbor.IsSolid() && !neighbor.IsTransparent() {
						continue
					}

					// Get texture index for this face
					var texIndex int
					switch face {
					case FaceTop:
						texIndex = blockInfo.TexTop
					case FaceBottom:
						texIndex = blockInfo.TexBottom
					default:
						texIndex = blockInfo.TexSide
					}

					texCoords := GetTexCoords(texIndex)
					normal := faceNormals[face]

					// World position for this block (float)
					fWorldX := float32(worldX)
					fWorldZ := float32(worldZ)

					// Add vertices for this face with AO
					for i := 0; i < 4; i++ {
						pos := faceVertices[face][i]

						// Calculate ambient occlusion
						ao := calculateAO(x, y, z, face, i, worldGetBlock, worldX, worldZ)

						// Position (3 floats) - use WORLD coordinates
						vertices = append(vertices, pos[0]+fWorldX, pos[1]+float32(y), pos[2]+fWorldZ)
						// TexCoord (2 floats)
						vertices = append(vertices, texCoords[i][0], texCoords[i][1])
						// Normal (3 floats)
						vertices = append(vertices, normal[0], normal[1], normal[2])
						// AO (1 float)
						vertices = append(vertices, ao)
						// Light (1 float)
						vertices = append(vertices, 1.0)
					}

					// Add indices
					indices = append(indices,
						indexOffset+0, indexOffset+1, indexOffset+2,
						indexOffset+2, indexOffset+3, indexOffset+0)
					indexOffset += 4
				}
			}
		}
	}

	// Upload to GPU
	if c.VAO == 0 {
		gl.GenVertexArrays(1, &c.VAO)
		gl.GenBuffers(1, &c.VBO)
		gl.GenBuffers(1, &c.EBO)
	}

	gl.BindVertexArray(c.VAO)

	gl.BindBuffer(gl.ARRAY_BUFFER, c.VBO)
	if len(vertices) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	}

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, c.EBO)
	if len(indices) > 0 {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)
	}

	// Vertex attributes
	stride := int32(10 * 4) // 10 floats per vertex

	// Position
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(0)

	// TexCoord
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 3*4)
	gl.EnableVertexAttribArray(1)

	// Normal
	gl.VertexAttribPointerWithOffset(2, 3, gl.FLOAT, false, stride, 5*4)
	gl.EnableVertexAttribArray(2)

	// AO
	gl.VertexAttribPointerWithOffset(3, 1, gl.FLOAT, false, stride, 8*4)
	gl.EnableVertexAttribArray(3)

	// Light
	gl.VertexAttribPointerWithOffset(4, 1, gl.FLOAT, false, stride, 9*4)
	gl.EnableVertexAttribArray(4)

	gl.BindVertexArray(0)

	c.IndexCount = int32(len(indices))
	c.MeshDirty = false
	c.MeshBuilt = true
}

// calculateAO calculates ambient occlusion for a vertex on any face
func calculateAO(x, y, z, face, vertex int, getBlock func(x, y, z int) BlockType, worldX, worldZ int) float32 {
	side1, side2, corner := false, false, false

	switch face {
	case FaceTop: // Y+
		switch vertex {
		case 0: // (0,1,0)
			side1 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ-1).IsSolid()
		case 1: // (1,1,0)
			side1 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ-1).IsSolid()
		case 2: // (1,1,1)
			side1 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ+1).IsSolid()
		case 3: // (0,1,1)
			side1 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ+1).IsSolid()
		}
	case FaceBottom: // Y-
		switch vertex {
		case 0: // (0,0,1)
			side1 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ+1).IsSolid()
		case 1: // (1,0,1)
			side1 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ+1).IsSolid()
		case 2: // (1,0,0)
			side1 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ-1).IsSolid()
		case 3: // (0,0,0)
			side1 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ-1).IsSolid()
		}
	case FaceNorth: // Z-
		switch vertex {
		case 0: // (1,0,0)
			side1 = getBlock(worldX+1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ-1).IsSolid()
		case 1: // (0,0,0)
			side1 = getBlock(worldX-1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ-1).IsSolid()
		case 2: // (0,1,0)
			side1 = getBlock(worldX-1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ-1).IsSolid()
		case 3: // (1,1,0)
			side1 = getBlock(worldX+1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ-1).IsSolid()
		}
	case FaceSouth: // Z+
		switch vertex {
		case 0: // (0,0,1)
			side1 = getBlock(worldX-1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ+1).IsSolid()
		case 1: // (1,0,1)
			side1 = getBlock(worldX+1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ+1).IsSolid()
		case 2: // (1,1,1)
			side1 = getBlock(worldX+1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ+1).IsSolid()
		case 3: // (0,1,1)
			side1 = getBlock(worldX-1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ+1).IsSolid()
		}
	case FaceEast: // X+
		switch vertex {
		case 0: // (1,0,1)
			side1 = getBlock(worldX+1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ+1).IsSolid()
		case 1: // (1,0,0)
			side1 = getBlock(worldX+1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ-1).IsSolid()
		case 2: // (1,1,0)
			side1 = getBlock(worldX+1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ-1).IsSolid()
		case 3: // (1,1,1)
			side1 = getBlock(worldX+1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ+1).IsSolid()
		}
	case FaceWest: // X-
		switch vertex {
		case 0: // (0,0,0)
			side1 = getBlock(worldX-1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ-1).IsSolid()
		case 1: // (0,0,1)
			side1 = getBlock(worldX-1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ+1).IsSolid()
		case 2: // (0,1,1)
			side1 = getBlock(worldX-1, y, worldZ+1).IsSolid()
			side2 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ+1).IsSolid()
		case 3: // (0,1,0)
			side1 = getBlock(worldX-1, y, worldZ-1).IsSolid()
			side2 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ-1).IsSolid()
		}
	}

	// Calculate AO value (0=fully occluded, 3=fully lit)
	ao := 0
	if side1 {
		ao++
	}
	if side2 {
		ao++
	}
	if corner && !side1 && !side2 {
		ao++
	}
	if side1 && side2 {
		ao = 3
	}

	aoValues := []float32{1.0, 0.75, 0.55, 0.35}
	return aoValues[ao]
}

// Render draws the chunk
func (c *Chunk) Render() {
	if !c.MeshBuilt || c.IndexCount == 0 {
		return
	}

	gl.BindVertexArray(c.VAO)
	gl.DrawElements(gl.TRIANGLES, c.IndexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
}

// Cleanup releases GPU resources
func (c *Chunk) Cleanup() {
	if c.VAO != 0 {
		gl.DeleteVertexArrays(1, &c.VAO)
		gl.DeleteBuffers(1, &c.VBO)
		gl.DeleteBuffers(1, &c.EBO)
		c.VAO = 0
		c.VBO = 0
		c.EBO = 0
	}
}
