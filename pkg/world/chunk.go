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

	// Solid mesh data
	VAO, VBO, EBO uint32
	IndexCount    int32

	// Transparent mesh data (water, glass, etc.)
	TransVAO, TransVBO, TransEBO uint32
	TransIndexCount              int32

	MeshDirty bool
	MeshBuilt bool

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
	// Top face (Y+) - CCW when viewed from above
	{{0, 1, 1}, {1, 1, 1}, {1, 1, 0}, {0, 1, 0}},
	// Bottom face (Y-) - CCW when viewed from below
	{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}},
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

	// Offset to prevent texture bleeding at distance with mipmaps
	offset := float32(0.002)

	return [4]mgl32.Vec2{
		{tx*TexCoordSize + offset, (ty+1)*TexCoordSize - offset},
		{(tx+1)*TexCoordSize - offset, (ty+1)*TexCoordSize - offset},
		{(tx+1)*TexCoordSize - offset, ty*TexCoordSize + offset},
		{tx*TexCoordSize + offset, ty*TexCoordSize + offset},
	}
}

// BuildMesh generates the mesh for the chunk, splitting solid and transparent geometry
func (c *Chunk) BuildMesh(getNeighborBlock func(x, y, z int) BlockType) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Separate buffers for solid and transparent blocks
	solidVerts := make([]float32, 0, 50000)
	solidIdx := make([]uint32, 0, 30000)
	var solidOff uint32

	transVerts := make([]float32, 0, 10000)
	transIdx := make([]uint32, 0, 6000)
	var transOff uint32

	// World-coordinate block lookup for cross-chunk boundaries and AO
	worldGetBlock := func(wx, wy, wz int) BlockType {
		lx := wx - c.X*ChunkSize
		lz := wz - c.Z*ChunkSize
		if lx >= 0 && lx < ChunkSize && lz >= 0 && lz < ChunkSize && wy >= 0 && wy < ChunkHeight {
			return c.Blocks[lx][wy][lz]
		}
		return getNeighborBlock(wx, wy, wz)
	}

	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < 128; y++ {
			for z := 0; z < ChunkSize; z++ {
				block := c.Blocks[x][y][z]
				if block == BlockAir {
					continue
				}

				blockInfo := BlockInfos[block]
				isTransparent := blockInfo.Transparent
				worldX := c.X*ChunkSize + x
				worldZ := c.Z*ChunkSize + z

				for face := 0; face < 6; face++ {
					var neighbor BlockType

					switch face {
					case 0:
						neighbor = worldGetBlock(worldX, y+1, worldZ)
					case 1:
						if y > 0 {
							neighbor = c.Blocks[x][y-1][z]
						}
					case 2:
						neighbor = worldGetBlock(worldX, y, worldZ-1)
					case 3:
						neighbor = worldGetBlock(worldX, y, worldZ+1)
					case 4:
						neighbor = worldGetBlock(worldX+1, y, worldZ)
					case 5:
						neighbor = worldGetBlock(worldX-1, y, worldZ)
					}

					// Determine if this face should be rendered
					if block == BlockWater {
						// Water: only render top face (surface) and edges next to non-water solid
						if neighbor == BlockWater {
							continue
						}
						if face == FaceTop && neighbor == BlockAir {
							// Water surface - render
						} else if face != FaceTop && neighbor == BlockAir {
							// Underwater side/bottom exposed to air (underwater caves) - skip
							continue
						} else if neighbor.IsSolid() && !neighbor.IsTransparent() {
							continue
						}
					} else if isTransparent {
						// Transparent blocks (glass, leaves): skip same-type neighbors
						if neighbor == block {
							continue
						}
						if neighbor.IsSolid() && !neighbor.IsTransparent() {
							continue
						}
					} else {
						// Solid blocks: skip if neighbor is solid and opaque
						if neighbor.IsSolid() && !neighbor.IsTransparent() {
							continue
						}
					}

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
					fWorldX := float32(worldX)
					fWorldZ := float32(worldZ)

					// Choose which buffer to append to
					verts := &solidVerts
					idx := &solidIdx
					off := &solidOff
					if isTransparent {
						verts = &transVerts
						idx = &transIdx
						off = &transOff
					}

					for i := 0; i < 4; i++ {
						pos := faceVertices[face][i]
						ao := float32(1.0)
						if !isTransparent {
							ao = calculateAO(x, y, z, face, i, worldGetBlock, worldX, worldZ)
						}
						*verts = append(*verts, pos[0]+fWorldX, pos[1]+float32(y), pos[2]+fWorldZ)
						*verts = append(*verts, texCoords[i][0], texCoords[i][1])
						*verts = append(*verts, normal[0], normal[1], normal[2])
						*verts = append(*verts, ao)
						*verts = append(*verts, 1.0)
					}

					*idx = append(*idx, *off+0, *off+1, *off+2, *off+2, *off+3, *off+0)
					*off += 4
				}
			}
		}
	}

	// Upload solid mesh
	if c.VAO == 0 {
		gl.GenVertexArrays(1, &c.VAO)
		gl.GenBuffers(1, &c.VBO)
		gl.GenBuffers(1, &c.EBO)
	}
	uploadMesh(c.VAO, c.VBO, c.EBO, solidVerts, solidIdx)
	c.IndexCount = int32(len(solidIdx))

	// Upload transparent mesh
	if c.TransVAO == 0 {
		gl.GenVertexArrays(1, &c.TransVAO)
		gl.GenBuffers(1, &c.TransVBO)
		gl.GenBuffers(1, &c.TransEBO)
	}
	uploadMesh(c.TransVAO, c.TransVBO, c.TransEBO, transVerts, transIdx)
	c.TransIndexCount = int32(len(transIdx))

	c.MeshDirty = false
	c.MeshBuilt = true
}

// uploadMesh uploads vertex/index data to a VAO
func uploadMesh(vao, vbo, ebo uint32, vertices []float32, indices []uint32) {
	gl.BindVertexArray(vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	if len(vertices) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	} else {
		gl.BufferData(gl.ARRAY_BUFFER, 0, nil, gl.STATIC_DRAW)
	}

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	if len(indices) > 0 {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)
	} else {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 0, nil, gl.STATIC_DRAW)
	}

	stride := int32(10 * 4)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 3*4)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(2, 3, gl.FLOAT, false, stride, 5*4)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(3, 1, gl.FLOAT, false, stride, 8*4)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(4, 1, gl.FLOAT, false, stride, 9*4)
	gl.EnableVertexAttribArray(4)

	gl.BindVertexArray(0)
}

// calculateAO calculates ambient occlusion for a vertex on any face
func calculateAO(x, y, z, face, vertex int, getBlock func(x, y, z int) BlockType, worldX, worldZ int) float32 {
	side1, side2, corner := false, false, false

	switch face {
	case FaceTop: // Y+ — vertices: (0,1,1), (1,1,1), (1,1,0), (0,1,0)
		switch vertex {
		case 0: // (0,1,1)
			side1 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ+1).IsSolid()
		case 1: // (1,1,1)
			side1 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ+1).IsSolid()
		case 2: // (1,1,0)
			side1 = getBlock(worldX+1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y+1, worldZ-1).IsSolid()
		case 3: // (0,1,0)
			side1 = getBlock(worldX-1, y+1, worldZ).IsSolid()
			side2 = getBlock(worldX, y+1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y+1, worldZ-1).IsSolid()
		}
	case FaceBottom: // Y- — vertices: (0,0,0), (1,0,0), (1,0,1), (0,0,1)
		switch vertex {
		case 0: // (0,0,0)
			side1 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ-1).IsSolid()
		case 1: // (1,0,0)
			side1 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ-1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ-1).IsSolid()
		case 2: // (1,0,1)
			side1 = getBlock(worldX+1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX+1, y-1, worldZ+1).IsSolid()
		case 3: // (0,0,1)
			side1 = getBlock(worldX-1, y-1, worldZ).IsSolid()
			side2 = getBlock(worldX, y-1, worldZ+1).IsSolid()
			corner = getBlock(worldX-1, y-1, worldZ+1).IsSolid()
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

// RenderSolid draws the solid (opaque) geometry
func (c *Chunk) RenderSolid() {
	if !c.MeshBuilt || c.IndexCount == 0 {
		return
	}
	gl.BindVertexArray(c.VAO)
	gl.DrawElements(gl.TRIANGLES, c.IndexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
}

// RenderTransparent draws the transparent geometry (water, glass, etc.)
func (c *Chunk) RenderTransparent() {
	if !c.MeshBuilt || c.TransIndexCount == 0 {
		return
	}
	gl.BindVertexArray(c.TransVAO)
	gl.DrawElements(gl.TRIANGLES, c.TransIndexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
}

// Render draws the chunk (backward compat — draws solid only)
func (c *Chunk) Render() {
	c.RenderSolid()
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
	if c.TransVAO != 0 {
		gl.DeleteVertexArrays(1, &c.TransVAO)
		gl.DeleteBuffers(1, &c.TransVBO)
		gl.DeleteBuffers(1, &c.TransEBO)
		c.TransVAO = 0
		c.TransVBO = 0
		c.TransEBO = 0
	}
}
