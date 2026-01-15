package engine

import (
	"fmt"
	"math"
	"runtime"

	"gocraft/pkg/player"
	"gocraft/pkg/render"
	"gocraft/pkg/ui"
	"gocraft/pkg/utils"
	"gocraft/pkg/world"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func init() {
	runtime.LockOSThread()
}

// Game represents the main game instance
type Game struct {
	window *glfw.Window
	width  int
	height int

	// Core systems
	world  *world.World
	player *player.Player
	input  *player.Input

	// Rendering
	blockShader   *render.Shader
	blockAtlas    *render.Texture
	sky           *render.Sky
	wireframe     *render.WireframeRenderer
	ui            *ui.UI

	// State
	running       bool
	paused        bool
	showDebug     bool
	cursorLocked  bool

	// Timing
	lastTime      float64
	deltaTime     float32
	fps           int
	frameCount    int
	fpsUpdateTime float64

	// Raycasting
	lookingAtBlock bool
	targetBlock    utils.Vec3i
	targetFace     int

	// Interaction cooldown
	lastBreak     float64
	lastPlace     float64
}

// NewGame creates a new game instance
func NewGame(width, height int, seed int64) (*Game, error) {
	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize GLFW: %w", err)
	}

	// Configure GLFW
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Samples, 4) // MSAA

	// Create window
	window, err := glfw.CreateWindow(width, height, "GoCraft - Minecraft in Go", nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("failed to create window: %w", err)
	}
	window.MakeContextCurrent()

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize OpenGL: %w", err)
	}

	// Print OpenGL info
	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Printf("OpenGL Version: %s\n", version)

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.MULTISAMPLE)

	// Create game instance
	game := &Game{
		window:       window,
		width:        width,
		height:       height,
		world:        world.NewWorld(seed, 8), // Render distance of 8 chunks
		player:       player.NewPlayer(),
		input:        player.NewInput(),
		running:      true,
		cursorLocked: true,
		showDebug:    true,
	}

	// Setup callbacks
	window.SetFramebufferSizeCallback(game.framebufferSizeCallback)
	window.SetKeyCallback(game.keyCallback)
	window.SetMouseButtonCallback(game.mouseButtonCallback)
	window.SetScrollCallback(game.scrollCallback)
	window.SetCursorPosCallback(game.cursorPosCallback)

	// Lock cursor
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	if glfw.RawMouseMotionSupported() {
		window.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}

	// Initialize rendering
	if err := game.initRendering(); err != nil {
		return nil, err
	}

	// Set player spawn position
	game.player.Position = game.world.GetSpawnPoint()

	return game, nil
}

func (g *Game) initRendering() error {
	var err error

	// Block shader
	g.blockShader, err = render.NewShader(render.BlockShaderVertex, render.BlockShaderFragment)
	if err != nil {
		return fmt.Errorf("failed to create block shader: %w", err)
	}

	// Generate block atlas
	g.blockAtlas = render.GenerateBlockAtlas()

	// Sky renderer
	g.sky, err = render.NewSky()
	if err != nil {
		return fmt.Errorf("failed to create sky renderer: %w", err)
	}

	// Wireframe renderer
	g.wireframe, err = render.NewWireframeRenderer()
	if err != nil {
		return fmt.Errorf("failed to create wireframe renderer: %w", err)
	}

	// UI
	g.ui, err = ui.NewUI(g.width, g.height)
	if err != nil {
		return fmt.Errorf("failed to create UI: %w", err)
	}

	return nil
}

// Run starts the game loop
func (g *Game) Run() {
	g.lastTime = glfw.GetTime()
	g.fpsUpdateTime = g.lastTime

	fmt.Println("Entering game loop...")

	for !g.window.ShouldClose() && g.running {
		// Calculate delta time
		currentTime := glfw.GetTime()
		g.deltaTime = float32(currentTime - g.lastTime)
		g.lastTime = currentTime

		// Cap delta time to prevent physics issues
		if g.deltaTime > 0.1 {
			g.deltaTime = 0.1
		}

		// FPS counter
		g.frameCount++
		if currentTime-g.fpsUpdateTime >= 1.0 {
			g.fps = g.frameCount
			g.frameCount = 0
			g.fpsUpdateTime = currentTime

			// Update window title
			title := fmt.Sprintf("GoCraft - FPS: %d | Chunks: %d | Pos: %.1f, %.1f, %.1f",
				g.fps, g.world.ChunkCount(),
				g.player.Position[0], g.player.Position[1], g.player.Position[2])
			g.window.SetTitle(title)
		}

		// Process input
		glfw.PollEvents()
		g.processInput()

		// Update
		g.update()

		// Render
		g.render()

		// Reset per-frame input
		g.input.Reset()

		// Swap buffers
		g.window.SwapBuffers()
	}
}

func (g *Game) processInput() {
	// Movement keys
	g.input.Forward = g.window.GetKey(glfw.KeyW) == glfw.Press
	g.input.Backward = g.window.GetKey(glfw.KeyS) == glfw.Press
	g.input.Left = g.window.GetKey(glfw.KeyA) == glfw.Press
	g.input.Right = g.window.GetKey(glfw.KeyD) == glfw.Press
	g.input.Jump = g.window.GetKey(glfw.KeySpace) == glfw.Press
	g.input.Sneak = g.window.GetKey(glfw.KeyLeftShift) == glfw.Press
	g.input.Sprint = g.window.GetKey(glfw.KeyLeftControl) == glfw.Press
}

func (g *Game) update() {
	// Update sky
	g.sky.Update(g.deltaTime)

	// Update player
	g.player.Update(g.deltaTime, g.world, g.input)

	// Update chunks
	g.world.UpdateChunks(g.player.Position[0], g.player.Position[2])

	// Build dirty meshes
	g.world.BuildDirtyMeshes()

	// Raycast for block selection
	ray := g.player.GetLookRay()
	result := g.world.Raycast(ray.Origin, ray.Direction, g.player.Reach)
	g.lookingAtBlock = result.Hit
	if result.Hit {
		g.targetBlock = result.BlockPos
		g.targetFace = result.Face
	}

	// Handle block interaction
	currentTime := glfw.GetTime()
	if g.input.LeftClick && g.lookingAtBlock {
		if currentTime-g.lastBreak > 0.2 {
			g.world.SetBlock(g.targetBlock.X, g.targetBlock.Y, g.targetBlock.Z, world.BlockAir)
			g.lastBreak = currentTime
		}
	}

	if g.input.RightClick && g.lookingAtBlock {
		if currentTime-g.lastPlace > 0.2 {
			// Calculate placement position based on face
			placePos := g.targetBlock
			switch g.targetFace {
			case 0: // X axis
				if g.player.GetForward()[0] > 0 {
					placePos.X--
				} else {
					placePos.X++
				}
			case 1: // Y axis
				if g.player.GetForward()[1] > 0 {
					placePos.Y--
				} else {
					placePos.Y++
				}
			case 2: // Z axis
				if g.player.GetForward()[2] > 0 {
					placePos.Z--
				} else {
					placePos.Z++
				}
			}

			// Check if placement position doesn't intersect player
			halfWidth := g.player.Width / 2
			playerAABB := utils.AABB{
				Min: mgl32.Vec3{g.player.Position[0] - halfWidth, g.player.Position[1], g.player.Position[2] - halfWidth},
				Max: mgl32.Vec3{g.player.Position[0] + halfWidth, g.player.Position[1] + g.player.Height, g.player.Position[2] + halfWidth},
			}
			blockAABB := utils.AABB{
				Min: mgl32.Vec3{float32(placePos.X), float32(placePos.Y), float32(placePos.Z)},
				Max: mgl32.Vec3{float32(placePos.X + 1), float32(placePos.Y + 1), float32(placePos.Z + 1)},
			}

			if !playerAABB.Intersects(blockAABB) {
				g.world.SetBlock(placePos.X, placePos.Y, placePos.Z, g.player.GetSelectedBlock())
				g.lastPlace = currentTime
			}
		}
	}
}

func (g *Game) render() {
	// Clear screen with blue to verify rendering works
	gl.ClearColor(0.3, 0.5, 0.8, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	fogColor := g.sky.GetFogColor()

	// Get view and projection matrices
	aspectRatio := float32(g.width) / float32(g.height)
	view := g.player.GetViewMatrix()
	projection := g.player.GetProjectionMatrix(aspectRatio)
	vp := projection.Mul4(view)

	// Render sky
	g.sky.Render(view, projection)

	// Render world
	gl.Enable(gl.DEPTH_TEST)

	g.blockShader.Use()
	g.blockShader.SetMat4("model", mgl32.Ident4())
	g.blockShader.SetMat4("view", view)
	g.blockShader.SetMat4("projection", projection)
	g.blockShader.SetVec3("cameraPos", g.player.GetEyePosition())
	g.blockShader.SetVec3("lightDir", g.sky.GetSunDirection())
	g.blockShader.SetVec3("lightColor", mgl32.Vec3{1.0, 0.95, 0.9})
	g.blockShader.SetVec3("ambientColor", mgl32.Vec3{0.3, 0.35, 0.4})
	g.blockShader.SetVec3("fogColor", fogColor)
	g.blockShader.SetFloat("fogStart", float32(g.world.RenderDist-2)*world.ChunkSize)
	g.blockShader.SetFloat("fogEnd", float32(g.world.RenderDist)*world.ChunkSize)
	g.blockShader.SetFloat("dayNightFactor", g.sky.GetDayNightFactor())
	g.blockShader.SetInt("textureSampler", 0)

	g.blockAtlas.Bind(0)

	// Frustum culling
	frustum := utils.ExtractFrustum(vp)
	g.world.Render(frustum)

	// Render block selection wireframe
	if g.lookingAtBlock {
		g.wireframe.Render(g.targetBlock.X, g.targetBlock.Y, g.targetBlock.Z,
			view, projection, mgl32.Vec4{0, 0, 0, 0.5})
	}

	// Render UI
	gl.Disable(gl.DEPTH_TEST)
	g.ui.RenderCrosshair()
	g.ui.RenderHotbar(g.player.Inventory, g.player.SelectedSlot, g.blockAtlas)
}

// Callbacks

func (g *Game) framebufferSizeCallback(window *glfw.Window, width, height int) {
	if width == 0 || height == 0 {
		return
	}
	g.width = width
	g.height = height
	gl.Viewport(0, 0, int32(width), int32(height))
	g.ui.Resize(width, height)
}

func (g *Game) keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		switch key {
		case glfw.KeyEscape:
			g.cursorLocked = !g.cursorLocked
			if g.cursorLocked {
				window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			} else {
				window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			}

		case glfw.KeyF:
			g.input.ToggleFly = true

		case glfw.KeyF3:
			g.showDebug = !g.showDebug

		case glfw.KeyQ:
			if mods&glfw.ModControl != 0 {
				g.running = false
			}

		// Number keys for hotbar selection
		case glfw.Key1:
			g.player.SelectSlot(0)
		case glfw.Key2:
			g.player.SelectSlot(1)
		case glfw.Key3:
			g.player.SelectSlot(2)
		case glfw.Key4:
			g.player.SelectSlot(3)
		case glfw.Key5:
			g.player.SelectSlot(4)
		case glfw.Key6:
			g.player.SelectSlot(5)
		case glfw.Key7:
			g.player.SelectSlot(6)
		case glfw.Key8:
			g.player.SelectSlot(7)
		case glfw.Key9:
			g.player.SelectSlot(8)
		}
	}
}

func (g *Game) mouseButtonCallback(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if !g.cursorLocked {
		if action == glfw.Press {
			g.cursorLocked = true
			window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		}
		return
	}

	if action == glfw.Press || action == glfw.Repeat {
		switch button {
		case glfw.MouseButtonLeft:
			g.input.LeftClick = true
		case glfw.MouseButtonRight:
			g.input.RightClick = true
		}
	}
}

func (g *Game) scrollCallback(window *glfw.Window, xoff, yoff float64) {
	g.input.ScrollDelta = float32(yoff)

	// Scroll through hotbar
	newSlot := g.player.SelectedSlot - int(yoff)
	newSlot = ((newSlot % 9) + 9) % 9 // Proper modulo for negative numbers
	g.player.SelectSlot(newSlot)
}

var lastX, lastY float64
var firstMouse = true

func (g *Game) cursorPosCallback(window *glfw.Window, xpos, ypos float64) {
	if !g.cursorLocked {
		firstMouse = true
		return
	}

	if firstMouse {
		lastX = xpos
		lastY = ypos
		firstMouse = false
		return
	}

	dx := float32(xpos - lastX)
	dy := float32(ypos - lastY)
	lastX = xpos
	lastY = ypos

	// Clamp large deltas (window refocus, etc.)
	maxDelta := float32(100)
	if math.Abs(float64(dx)) > float64(maxDelta) || math.Abs(float64(dy)) > float64(maxDelta) {
		return
	}

	g.input.MouseDeltaX = dx
	g.input.MouseDeltaY = dy
}

// Cleanup releases all resources
func (g *Game) Cleanup() {
	g.world.Cleanup()
	g.blockShader.Delete()
	g.blockAtlas.Delete()
	g.sky.Cleanup()
	g.wireframe.Cleanup()
	g.ui.Cleanup()
	glfw.Terminate()
}
