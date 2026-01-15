package world

import (
	"fmt"
	"math"
	"sync"

	"gocraft/pkg/utils"

	"github.com/go-gl/mathgl/mgl32"
)

// World represents the game world
type World struct {
	Chunks       map[string]*Chunk
	ChunksMutex  sync.RWMutex
	Generator    *WorldGenerator
	RenderDist   int
	LoadQueue    chan ChunkPos
	UnloadQueue  chan ChunkPos
	PlayerChunkX int
	PlayerChunkZ int
}

// ChunkPos represents a chunk position
type ChunkPos struct {
	X, Z int
}

// NewWorld creates a new world
func NewWorld(seed int64, renderDist int) *World {
	return &World{
		Chunks:      make(map[string]*Chunk),
		Generator:   NewWorldGenerator(seed),
		RenderDist:  renderDist,
		LoadQueue:   make(chan ChunkPos, 256),
		UnloadQueue: make(chan ChunkPos, 256),
	}
}

// chunkKey returns a string key for a chunk position
func chunkKey(x, z int) string {
	return fmt.Sprintf("%d,%d", x, z)
}

// GetChunk returns the chunk at the given chunk coordinates
func (w *World) GetChunk(cx, cz int) *Chunk {
	w.ChunksMutex.RLock()
	defer w.ChunksMutex.RUnlock()
	return w.Chunks[chunkKey(cx, cz)]
}

// GetOrCreateChunk gets or creates a chunk
func (w *World) GetOrCreateChunk(cx, cz int) *Chunk {
	key := chunkKey(cx, cz)

	w.ChunksMutex.RLock()
	chunk := w.Chunks[key]
	w.ChunksMutex.RUnlock()

	if chunk != nil {
		return chunk
	}

	// Create new chunk
	chunk = NewChunk(cx, cz)
	w.Generator.GenerateChunk(chunk)

	w.ChunksMutex.Lock()
	w.Chunks[key] = chunk
	w.ChunksMutex.Unlock()

	return chunk
}

// GetBlock returns the block at world coordinates
func (w *World) GetBlock(x, y, z int) BlockType {
	if y < 0 || y >= ChunkHeight {
		return BlockAir
	}

	cx := x >> 4 // Divide by 16
	cz := z >> 4
	lx := utils.Mod(x, ChunkSize)
	lz := utils.Mod(z, ChunkSize)

	chunk := w.GetChunk(cx, cz)
	if chunk == nil {
		return BlockAir
	}

	return chunk.GetBlock(lx, y, lz)
}

// SetBlock sets a block at world coordinates
func (w *World) SetBlock(x, y, z int, block BlockType) {
	if y < 0 || y >= ChunkHeight {
		return
	}

	cx := x >> 4
	cz := z >> 4
	lx := utils.Mod(x, ChunkSize)
	lz := utils.Mod(z, ChunkSize)

	chunk := w.GetChunk(cx, cz)
	if chunk == nil {
		return
	}

	chunk.SetBlock(lx, y, lz, block)

	// Mark neighboring chunks as dirty if on chunk boundary
	if lx == 0 {
		if neighbor := w.GetChunk(cx-1, cz); neighbor != nil {
			neighbor.MeshDirty = true
		}
	} else if lx == ChunkSize-1 {
		if neighbor := w.GetChunk(cx+1, cz); neighbor != nil {
			neighbor.MeshDirty = true
		}
	}
	if lz == 0 {
		if neighbor := w.GetChunk(cx, cz-1); neighbor != nil {
			neighbor.MeshDirty = true
		}
	} else if lz == ChunkSize-1 {
		if neighbor := w.GetChunk(cx, cz+1); neighbor != nil {
			neighbor.MeshDirty = true
		}
	}
}

// UpdateChunks loads/unloads chunks based on player position
func (w *World) UpdateChunks(playerX, playerZ float32) {
	pcx := int(math.Floor(float64(playerX) / ChunkSize))
	pcz := int(math.Floor(float64(playerZ) / ChunkSize))

	w.PlayerChunkX = pcx
	w.PlayerChunkZ = pcz

	// Limit chunks loaded per frame to prevent freezing
	chunksLoadedThisFrame := 0
	maxChunksPerFrame := 2

	// Load chunks in render distance
	for cx := pcx - w.RenderDist; cx <= pcx+w.RenderDist; cx++ {
		for cz := pcz - w.RenderDist; cz <= pcz+w.RenderDist; cz++ {
			// Check if chunk is within circular render distance
			dx := cx - pcx
			dz := cz - pcz
			if dx*dx+dz*dz > w.RenderDist*w.RenderDist {
				continue
			}

			if w.GetChunk(cx, cz) == nil {
				if chunksLoadedThisFrame >= maxChunksPerFrame {
					return // Load more next frame
				}
				w.GetOrCreateChunk(cx, cz)
				chunksLoadedThisFrame++
			}
		}
	}

	// Unload far chunks
	w.ChunksMutex.Lock()
	for key, chunk := range w.Chunks {
		dx := chunk.X - pcx
		dz := chunk.Z - pcz
		if dx*dx+dz*dz > (w.RenderDist+2)*(w.RenderDist+2) {
			chunk.Cleanup()
			delete(w.Chunks, key)
		}
	}
	w.ChunksMutex.Unlock()
}

// BuildDirtyMeshes rebuilds meshes for chunks that need it
func (w *World) BuildDirtyMeshes() {
	w.ChunksMutex.RLock()
	defer w.ChunksMutex.RUnlock()

	getBlock := func(x, y, z int) BlockType {
		return w.GetBlock(x, y, z)
	}

	// Build a few meshes per frame
	built := 0
	for _, chunk := range w.Chunks {
		if chunk.MeshDirty {
			chunk.BuildMesh(getBlock)
			built++
			if built >= 2 {
				return
			}
		}
	}
}

// Render renders all visible chunks
func (w *World) Render(frustum utils.Frustum) int {
	w.ChunksMutex.RLock()
	defer w.ChunksMutex.RUnlock()

	rendered := 0
	for _, chunk := range w.Chunks {
		// Frustum culling
		if !frustum.ContainsAABB(chunk.GetAABB()) {
			continue
		}

		chunk.Render()
		rendered++
	}

	return rendered
}

// Raycast performs a simple raycast in the world
func (w *World) Raycast(origin, direction mgl32.Vec3, maxDist float32) utils.RaycastResult {
	result := utils.RaycastResult{}

	// Simple step-based raycasting
	step := float32(0.1)

	for d := float32(0); d < maxDist; d += step {
		pos := origin.Add(direction.Mul(d))

		x := int(math.Floor(float64(pos[0])))
		y := int(math.Floor(float64(pos[1])))
		z := int(math.Floor(float64(pos[2])))

		block := w.GetBlock(x, y, z)
		if block != BlockAir && block.IsSolid() {
			result.Hit = true
			result.BlockPos = utils.Vec3i{X: x, Y: y, Z: z}
			result.Distance = d
			result.Position = pos
			result.Face = 0
			result.Normal = mgl32.Vec3{0, 1, 0}
			return result
		}
	}

	return result
}

// GetSpawnPoint returns a suitable spawn point
func (w *World) GetSpawnPoint() mgl32.Vec3 {
	x, y, z := w.Generator.GetSpawnPoint()
	return mgl32.Vec3{float32(x), float32(y), float32(z)}
}

// ChunkCount returns the number of loaded chunks
func (w *World) ChunkCount() int {
	w.ChunksMutex.RLock()
	defer w.ChunksMutex.RUnlock()
	return len(w.Chunks)
}

// Cleanup releases all resources
func (w *World) Cleanup() {
	w.ChunksMutex.Lock()
	defer w.ChunksMutex.Unlock()

	for _, chunk := range w.Chunks {
		chunk.Cleanup()
	}
	w.Chunks = make(map[string]*Chunk)
}
