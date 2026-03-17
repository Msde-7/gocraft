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
	font         *BitmapFont
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

	ui.font = NewBitmapFont()

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

	slotSize := float32(48)
	padding := float32(4)
	totalWidth := 9*slotSize + 8*padding
	startX := (float32(ui.screenWidth) - totalWidth) / 2
	startY := float32(ui.screenHeight) - slotSize - 10

	// Draw slot backgrounds
	for i := 0; i < 9; i++ {
		x := startX + float32(i)*(slotSize+padding)

		// Slot border and background
		if i == selectedSlot {
			// Draw selection border
			ui.shader.SetVec4("color", mgl32.Vec4{1, 1, 1, 0.9})
			ui.shader.SetInt("useTexture", 0)
			ui.drawRect(x-2, startY-2, slotSize+4, slotSize+4)
			// Inner background
			ui.shader.SetVec4("color", mgl32.Vec4{0.4, 0.4, 0.4, 0.85})
			ui.drawRect(x, startY, slotSize, slotSize)
		} else {
			ui.shader.SetVec4("color", mgl32.Vec4{0.15, 0.15, 0.15, 0.75})
			ui.shader.SetInt("useTexture", 0)
			ui.drawRect(x, startY, slotSize, slotSize)
		}

		// Draw slot number
		ui.drawText(fmt.Sprintf("%d", i+1), x+2, startY+2, 1, mgl32.Vec4{0.7, 0.7, 0.7, 0.6})

		// Draw block icon
		block := inventory[i]
		if block != world.BlockAir {
			ui.shader.Use()
			ui.shader.SetMat4("projection", ui.projection)
			ui.shader.SetVec4("color", mgl32.Vec4{1, 1, 1, 1})
			ui.shader.SetInt("useTexture", 1)
			blockAtlas.Bind(0)

			// Get texture coords for this block's top face
			texIndex := world.BlockInfos[block].TexTop
			ui.drawTexturedRect(x+6, startY+6, slotSize-12, slotSize-12, texIndex)
		}
	}

	// Draw selected block name above hotbar
	selectedBlock := inventory[selectedSlot]
	if selectedBlock != world.BlockAir {
		name := selectedBlock.GetName()
		scale := float32(2)
		textW := float32(len(name)) * 6 * scale
		textX := float32(ui.screenWidth)/2 - textW/2
		textY := startY - 28

		// Background behind text
		ui.shader.Use()
		ui.shader.SetMat4("projection", ui.projection)
		ui.shader.SetVec4("color", mgl32.Vec4{0, 0, 0, 0.5})
		ui.shader.SetInt("useTexture", 0)
		ui.drawRect(textX-4, textY-2, textW+8, 7*scale+4)

		// Text
		ui.drawText(name, textX, textY, scale, mgl32.Vec4{1, 1, 1, 1})
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

// drawText renders a string using the bitmap font
func (ui *UI) drawText(text string, x, y, scale float32, col mgl32.Vec4) {
	ui.shader.Use()
	ui.shader.SetMat4("projection", ui.projection)
	ui.shader.SetVec4("color", col)
	ui.shader.SetInt("useTexture", 1)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, ui.font.textureID)

	charW := float32(ui.font.charWidth)
	charH := float32(ui.font.charHeight)
	cols := float32(ui.font.cols)
	rows := float32(6)
	texW := 1.0 / cols
	texH := 1.0 / rows

	for i, ch := range text {
		if ch < 32 || ch > 127 {
			ch = '?'
		}
		idx := int(ch) - 32
		tc := float32(idx%int(cols)) * texW
		tr := float32(idx/int(cols)) * texH

		dx := x + float32(i)*(charW+1)*scale
		dy := y

		vertices := []float32{
			dx, dy, tc, tr,
			dx + charW*scale, dy, tc + texW, tr,
			dx + charW*scale, dy + charH*scale, tc + texW, tr + texH,
			dx, dy, tc, tr,
			dx + charW*scale, dy + charH*scale, tc + texW, tr + texH,
			dx, dy + charH*scale, tc, tr + texH,
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
	ui.font.Delete()
	ui.shader.Delete()
}
