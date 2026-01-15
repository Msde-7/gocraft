package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// WireframeRenderer renders wireframe outlines for blocks
type WireframeRenderer struct {
	shader   *Shader
	vao, vbo uint32
}

// NewWireframeRenderer creates a new wireframe renderer
func NewWireframeRenderer() (*WireframeRenderer, error) {
	shader, err := NewShader(WireframeShaderVertex, WireframeShaderFragment)
	if err != nil {
		return nil, err
	}

	w := &WireframeRenderer{
		shader: shader,
	}

	// Create cube wireframe vertices
	vertices := []float32{
		// Bottom face
		0, 0, 0, 1, 0, 0,
		1, 0, 0, 1, 0, 1,
		1, 0, 1, 0, 0, 1,
		0, 0, 1, 0, 0, 0,
		// Top face
		0, 1, 0, 1, 1, 0,
		1, 1, 0, 1, 1, 1,
		1, 1, 1, 0, 1, 1,
		0, 1, 1, 0, 1, 0,
		// Vertical edges
		0, 0, 0, 0, 1, 0,
		1, 0, 0, 1, 1, 0,
		1, 0, 1, 1, 1, 1,
		0, 0, 1, 0, 1, 1,
	}

	gl.GenVertexArrays(1, &w.vao)
	gl.GenBuffers(1, &w.vbo)

	gl.BindVertexArray(w.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*4, 0)
	gl.EnableVertexAttribArray(0)

	gl.BindVertexArray(0)

	return w, nil
}

// Render renders a wireframe cube at the given position
func (w *WireframeRenderer) Render(x, y, z int, view, projection mgl32.Mat4, color mgl32.Vec4) {
	w.shader.Use()

	// Create model matrix with slight expansion to avoid z-fighting
	expansion := float32(0.002)
	model := mgl32.Translate3D(float32(x)-expansion, float32(y)-expansion, float32(z)-expansion)
	model = model.Mul4(mgl32.Scale3D(1+expansion*2, 1+expansion*2, 1+expansion*2))

	w.shader.SetMat4("model", model)
	w.shader.SetMat4("view", view)
	w.shader.SetMat4("projection", projection)
	w.shader.SetVec4("color", color)

	gl.LineWidth(2.0)
	gl.BindVertexArray(w.vao)
	gl.DrawArrays(gl.LINES, 0, 24)
	gl.BindVertexArray(0)
	gl.LineWidth(1.0)
}

// Cleanup releases resources
func (w *WireframeRenderer) Cleanup() {
	gl.DeleteVertexArrays(1, &w.vao)
	gl.DeleteBuffers(1, &w.vbo)
	w.shader.Delete()
}
