package ui

import (
	"fmt"

	"gocraft/pkg/render"
	"gocraft/pkg/world"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// UI handles all user interface rendering
type UI struct {
	shader       *render.Shader
	vao, vbo     uint32
	projection   mgl32.Mat4
	screenWidth  int
	screenHeight int
}

// NewUI creates a new UI renderer
func NewUI(width, height int) (*UI, error) {
	shader, err := render.NewShader(render.UIShaderVertex, render.UIShaderFragment)
	if err != nil {
		return nil, err
	}

	ui := &UI{
		shader:       shader,
		screenWidth:  width,
		screenHeight: height,
	}

	gl.GenVertexArrays(1, &ui.vao)
	gl.GenBuffers(1, &ui.vbo)

	ui.updateProjection()
	return ui, nil
}

func (ui *UI) updateProjection() {
	ui.projection = mgl32.Ortho(0, float32(ui.screenWidth), float32(ui.screenHeight), 0, -1, 1)
}

// Resize updates the UI for new screen dimensions
func (ui *UI) Resize(width, height int) {
	ui.screenWidth = width
	ui.screenHeight = height
	ui.updateProjection()
}

// RenderCrosshair renders the crosshair in the center of the screen
func (ui *UI) RenderCrosshair() {
	ui.shader.Use()
	ui.shader.SetMat4("projection", ui.projection)
	ui.shader.SetVec4("color", mgl32.Vec4{1, 1, 1, 0.8})
	ui.shader.SetInt("useTexture", 0)

	cx := float32(ui.screenWidth) / 2
	cy := float32(ui.screenHeight) / 2
	size := float32(12)
	thickness := float32(2)

	// Horizontal line
	ui.drawRect(cx-size, cy-thickness/2, size*2, thickness)
	// Vertical line
	ui.drawRect(cx-thickness/2, cy-size, thickness, size*2)
}

// RenderHotbar renders the hotbar at the bottom of the screen
func (ui *UI) RenderHotbar(inventory [9]world.BlockType, selectedSlot int, blockAtlas *render.Texture) {
	ui.shader.Use()
	ui.shader.SetMat4("projection", ui.projection)

	slotSize := float32(40)
	padding := float32(4)
	totalWidth := 9*slotSize + 8*padding
	startX := (float32(ui.screenWidth) - totalWidth) / 2
	startY := float32(ui.screenHeight) - slotSize - 20

	// Draw slot backgrounds
	for i := 0; i < 9; i++ {
		x := startX + float32(i)*(slotSize+padding)

		// Slot background
		if i == selectedSlot {
			ui.shader.SetVec4("color", mgl32.Vec4{0.8, 0.8, 0.8, 0.8})
		} else {
			ui.shader.SetVec4("color", mgl32.Vec4{0.2, 0.2, 0.2, 0.7})
		}
		ui.shader.SetInt("useTexture", 0)
		ui.drawRect(x, startY, slotSize, slotSize)

		// Draw block icon
		block := inventory[i]
		if block != world.BlockAir {
			ui.shader.SetVec4("color", mgl32.Vec4{1, 1, 1, 1})
			ui.shader.SetInt("useTexture", 1)
			blockAtlas.Bind(0)

			// Get texture coords for this block's top face
			texIndex := world.BlockInfos[block].TexTop
			ui.drawTexturedRect(x+4, startY+4, slotSize-8, slotSize-8, texIndex)
		}

	}
}

// RenderBlockSelector renders the selected block info
func (ui *UI) RenderBlockSelector(block world.BlockType) {
	ui.shader.Use()
	ui.shader.SetMat4("projection", ui.projection)
	ui.shader.SetVec4("color", mgl32.Vec4{0, 0, 0, 0.5})
	ui.shader.SetInt("useTexture", 0)

	// Background for block name
	y := float32(ui.screenHeight) - 80
	ui.drawRect(float32(ui.screenWidth)/2-60, y, 120, 20)
}

// RenderDebugInfo renders debug information
func (ui *UI) RenderDebugInfo(fps int, pos mgl32.Vec3, chunks int) {
	// In a full implementation, we'd render text here
	// For now, this is a placeholder
	_ = fmt.Sprintf("FPS: %d | Pos: %.1f, %.1f, %.1f | Chunks: %d",
		fps, pos[0], pos[1], pos[2], chunks)
}

// drawRect draws a rectangle
func (ui *UI) drawRect(x, y, w, h float32) {
	vertices := []float32{
		x, y, 0, 0,
		x + w, y, 1, 0,
		x + w, y + h, 1, 1,
		x, y, 0, 0,
		x + w, y + h, 1, 1,
		x, y + h, 0, 1,
	}

	gl.BindVertexArray(ui.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, ui.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)
	gl.EnableVertexAttribArray(1)

	gl.DrawArrays(gl.TRIANGLES, 0, 6)
	gl.BindVertexArray(0)
}

// drawTexturedRect draws a textured rectangle from the atlas
func (ui *UI) drawTexturedRect(x, y, w, h float32, texIndex int) {
	atlasSize := float32(16)
	texCoordSize := float32(1.0 / atlasSize)

	tx := float32(texIndex%16) * texCoordSize
	ty := float32(texIndex/16) * texCoordSize

	vertices := []float32{
		x, y, tx, ty,
		x + w, y, tx + texCoordSize, ty,
		x + w, y + h, tx + texCoordSize, ty + texCoordSize,
		x, y, tx, ty,
		x + w, y + h, tx + texCoordSize, ty + texCoordSize,
		x, y + h, tx, ty + texCoordSize,
	}

	gl.BindVertexArray(ui.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, ui.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)
	gl.EnableVertexAttribArray(1)

	gl.DrawArrays(gl.TRIANGLES, 0, 6)
	gl.BindVertexArray(0)
}

// Cleanup releases resources
func (ui *UI) Cleanup() {
	gl.DeleteVertexArrays(1, &ui.vao)
	gl.DeleteBuffers(1, &ui.vbo)
	ui.shader.Delete()
}
