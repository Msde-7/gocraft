package player

import (
	"math"

	"gocraft/pkg/utils"
	"gocraft/pkg/world"

	"github.com/go-gl/mathgl/mgl32"
)

// Player represents the player in the game
type Player struct {
	Position mgl32.Vec3
	Velocity mgl32.Vec3
	Rotation mgl32.Vec2 // Yaw, Pitch

	// Movement
	Speed       float32
	JumpForce   float32
	OnGround    bool
	Flying      bool
	Sprinting   bool
	Sneaking    bool
	NoClip      bool

	// Physics
	Width       float32
	Height      float32
	EyeHeight   float32
	Gravity     float32

	// Camera
	FOV         float32
	Near        float32
	Far         float32

	// Interaction
	Reach        float32
	SelectedSlot int
	Inventory    [9]world.BlockType
}

// NewPlayer creates a new player
func NewPlayer() *Player {
	p := &Player{
		Position:  mgl32.Vec3{0, 80, 0},
		Rotation:  mgl32.Vec2{0, 0},
		Speed:     6.0,
		JumpForce: 8.5,
		Width:     0.6,
		Height:    1.8,
		EyeHeight: 1.62,
		Gravity:   28.0,
		FOV:       70.0,
		Near:      0.1,
		Far:       500.0,
		Reach:     5.0,
		Flying:    true, // Start in fly mode
	}

	// Initialize inventory with some blocks
	p.Inventory = [9]world.BlockType{
		world.BlockGrass,
		world.BlockDirt,
		world.BlockStone,
		world.BlockCobblestone,
		world.BlockOakPlanks,
		world.BlockGlass,
		world.BlockBrick,
		world.BlockWood,
		world.BlockSand,
	}

	return p
}

// GetEyePosition returns the position of the player's eyes
func (p *Player) GetEyePosition() mgl32.Vec3 {
	return p.Position.Add(mgl32.Vec3{0, p.EyeHeight, 0})
}

// GetForward returns the forward direction vector
func (p *Player) GetForward() mgl32.Vec3 {
	yaw := p.Rotation[0] * math.Pi / 180.0
	pitch := p.Rotation[1] * math.Pi / 180.0

	return mgl32.Vec3{
		float32(-math.Sin(float64(yaw)) * math.Cos(float64(pitch))),
		float32(-math.Sin(float64(pitch))),
		float32(-math.Cos(float64(yaw)) * math.Cos(float64(pitch))),
	}
}

// GetRight returns the right direction vector
func (p *Player) GetRight() mgl32.Vec3 {
	yaw := p.Rotation[0] * math.Pi / 180.0
	return mgl32.Vec3{
		float32(math.Cos(float64(yaw))),
		0,
		float32(-math.Sin(float64(yaw))),
	}
}

// GetViewMatrix returns the view matrix for rendering
func (p *Player) GetViewMatrix() mgl32.Mat4 {
	eye := p.GetEyePosition()
	forward := p.GetForward()
	target := eye.Add(forward)
	up := mgl32.Vec3{0, 1, 0}

	return mgl32.LookAtV(eye, target, up)
}

// GetProjectionMatrix returns the projection matrix
func (p *Player) GetProjectionMatrix(aspectRatio float32) mgl32.Mat4 {
	fov := p.FOV
	if p.Sprinting {
		fov += 10
	}
	return mgl32.Perspective(mgl32.DegToRad(fov), aspectRatio, p.Near, p.Far)
}

// Update updates the player state
func (p *Player) Update(dt float32, w *world.World, input *Input) {
	// Handle rotation - subtract X for correct direction
	p.Rotation[0] -= input.MouseDeltaX * input.Sensitivity
	p.Rotation[1] += input.MouseDeltaY * input.Sensitivity

	// Clamp pitch
	if p.Rotation[1] > 89 {
		p.Rotation[1] = 89
	}
	if p.Rotation[1] < -89 {
		p.Rotation[1] = -89
	}

	// Handle movement input
	moveDir := mgl32.Vec3{}
	forward := p.GetForward()
	right := p.GetRight()

	// Only use horizontal component for walking
	forward[1] = 0
	forward = forward.Normalize()

	if input.Forward {
		moveDir = moveDir.Add(forward)
	}
	if input.Backward {
		moveDir = moveDir.Sub(forward)
	}
	if input.Left {
		moveDir = moveDir.Sub(right)
	}
	if input.Right {
		moveDir = moveDir.Add(right)
	}

	// Normalize movement direction
	if moveDir.Len() > 0 {
		moveDir = moveDir.Normalize()
	}

	// Calculate speed
	speed := p.Speed
	if p.Sprinting && input.Forward {
		speed *= 1.5
	}
	if p.Sneaking {
		speed *= 0.3
	}
	if p.Flying {
		speed *= 2.5
	}

	if p.Flying || p.NoClip {
		// Flying movement
		if input.Jump {
			p.Velocity[1] = speed
		} else if input.Sneak {
			p.Velocity[1] = -speed
		} else {
			p.Velocity[1] = 0
		}

		p.Velocity[0] = moveDir[0] * speed
		p.Velocity[2] = moveDir[2] * speed
	} else {
		// Ground movement
		p.Velocity[0] = moveDir[0] * speed
		p.Velocity[2] = moveDir[2] * speed

		// Apply gravity
		p.Velocity[1] -= p.Gravity * dt

		// Cap fall speed
		if p.Velocity[1] < -50 {
			p.Velocity[1] = -50
		}

		// Jump
		if input.Jump && p.OnGround {
			p.Velocity[1] = p.JumpForce
			p.OnGround = false
		}
	}

	// Handle special keys
	if input.ToggleFly {
		p.Flying = !p.Flying
		if p.Flying {
			p.Velocity[1] = 0
		}
	}

	p.Sprinting = input.Sprint
	p.Sneaking = input.Sneak

	// Apply movement with collision
	if p.NoClip {
		p.Position = p.Position.Add(p.Velocity.Mul(dt))
	} else {
		p.move(dt, w)
	}
}

// move applies movement with collision detection
func (p *Player) move(dt float32, w *world.World) {
	// Move each axis separately for proper collision response
	halfWidth := p.Width / 2

	// Move X
	newPos := p.Position
	newPos[0] += p.Velocity[0] * dt

	if !p.checkCollision(newPos, halfWidth, p.Height, w) {
		p.Position[0] = newPos[0]
	} else {
		p.Velocity[0] = 0
	}

	// Move Y
	newPos = p.Position
	newPos[1] += p.Velocity[1] * dt

	wasOnGround := p.OnGround
	if !p.checkCollision(newPos, halfWidth, p.Height, w) {
		p.Position[1] = newPos[1]
		p.OnGround = false
	} else {
		if p.Velocity[1] < 0 {
			// Landing
			p.OnGround = true
			// Snap to ground
			p.Position[1] = float32(math.Floor(float64(p.Position[1])))
		}
		p.Velocity[1] = 0
	}

	// Check if we're standing on ground (for the next frame)
	if !p.OnGround && !wasOnGround {
		checkPos := p.Position
		checkPos[1] -= 0.1
		if p.checkCollision(checkPos, halfWidth, p.Height, w) {
			p.OnGround = true
		}
	}

	// Move Z
	newPos = p.Position
	newPos[2] += p.Velocity[2] * dt

	if !p.checkCollision(newPos, halfWidth, p.Height, w) {
		p.Position[2] = newPos[2]
	} else {
		p.Velocity[2] = 0
	}
}

// checkCollision checks if the player would collide at the given position
func (p *Player) checkCollision(pos mgl32.Vec3, halfWidth, height float32, w *world.World) bool {
	// Check all blocks the player could be touching
	minX := int(math.Floor(float64(pos[0] - halfWidth)))
	maxX := int(math.Floor(float64(pos[0] + halfWidth)))
	minY := int(math.Floor(float64(pos[1])))
	maxY := int(math.Floor(float64(pos[1] + height)))
	minZ := int(math.Floor(float64(pos[2] - halfWidth)))
	maxZ := int(math.Floor(float64(pos[2] + halfWidth)))

	playerAABB := utils.AABB{
		Min: mgl32.Vec3{pos[0] - halfWidth, pos[1], pos[2] - halfWidth},
		Max: mgl32.Vec3{pos[0] + halfWidth, pos[1] + height, pos[2] + halfWidth},
	}

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				block := w.GetBlock(x, y, z)
				if !block.IsSolid() {
					continue
				}

				blockAABB := utils.AABB{
					Min: mgl32.Vec3{float32(x), float32(y), float32(z)},
					Max: mgl32.Vec3{float32(x + 1), float32(y + 1), float32(z + 1)},
				}

				if playerAABB.Intersects(blockAABB) {
					return true
				}
			}
		}
	}

	return false
}

// GetLookRay returns a ray from the player's eyes in the look direction
func (p *Player) GetLookRay() utils.Ray {
	return utils.NewRay(p.GetEyePosition(), p.GetForward())
}

// GetSelectedBlock returns the currently selected block type
func (p *Player) GetSelectedBlock() world.BlockType {
	return p.Inventory[p.SelectedSlot]
}

// SelectSlot selects a hotbar slot
func (p *Player) SelectSlot(slot int) {
	if slot >= 0 && slot < 9 {
		p.SelectedSlot = slot
	}
}

// Input represents player input state
type Input struct {
	Forward     bool
	Backward    bool
	Left        bool
	Right       bool
	Jump        bool
	Sneak       bool
	Sprint      bool
	ToggleFly   bool
	MouseDeltaX float32
	MouseDeltaY float32
	Sensitivity float32
	LeftClick   bool
	RightClick  bool
	ScrollDelta float32
}

// NewInput creates a new input state
func NewInput() *Input {
	return &Input{
		Sensitivity: 0.5, // Camera sensitivity
	}
}

// Reset resets per-frame input state
func (i *Input) Reset() {
	i.MouseDeltaX = 0
	i.MouseDeltaY = 0
	i.ToggleFly = false
	i.LeftClick = false
	i.RightClick = false
	i.ScrollDelta = 0
}
