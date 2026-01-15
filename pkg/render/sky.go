package render

import (
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Sky renders the skybox and handles day/night cycle
type Sky struct {
	shader     *Shader
	vao        uint32
	vbo        uint32
	TimeOfDay  float32 // 0-24 hours
	DayLength  float32 // Real seconds per game day
}

// NewSky creates a new sky renderer
func NewSky() (*Sky, error) {
	shader, err := NewShader(SkyShaderVertex, SkyShaderFragment)
	if err != nil {
		return nil, err
	}

	sky := &Sky{
		shader:    shader,
		TimeOfDay: 8.0, // Start at 8 AM
		DayLength: 600, // 10 minute days
	}

	sky.createMesh()
	return sky, nil
}

func (s *Sky) createMesh() {
	// Cube vertices for skybox
	vertices := []float32{
		// Back face
		-1.0, -1.0, -1.0,
		1.0, 1.0, -1.0,
		1.0, -1.0, -1.0,
		1.0, 1.0, -1.0,
		-1.0, -1.0, -1.0,
		-1.0, 1.0, -1.0,
		// Front face
		-1.0, -1.0, 1.0,
		1.0, -1.0, 1.0,
		1.0, 1.0, 1.0,
		1.0, 1.0, 1.0,
		-1.0, 1.0, 1.0,
		-1.0, -1.0, 1.0,
		// Left face
		-1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0,
		-1.0, -1.0, -1.0,
		-1.0, -1.0, -1.0,
		-1.0, -1.0, 1.0,
		-1.0, 1.0, 1.0,
		// Right face
		1.0, 1.0, 1.0,
		1.0, -1.0, -1.0,
		1.0, 1.0, -1.0,
		1.0, -1.0, -1.0,
		1.0, 1.0, 1.0,
		1.0, -1.0, 1.0,
		// Bottom face
		-1.0, -1.0, -1.0,
		1.0, -1.0, -1.0,
		1.0, -1.0, 1.0,
		1.0, -1.0, 1.0,
		-1.0, -1.0, 1.0,
		-1.0, -1.0, -1.0,
		// Top face
		-1.0, 1.0, -1.0,
		1.0, 1.0, 1.0,
		1.0, 1.0, -1.0,
		1.0, 1.0, 1.0,
		-1.0, 1.0, -1.0,
		-1.0, 1.0, 1.0,
	}

	gl.GenVertexArrays(1, &s.vao)
	gl.GenBuffers(1, &s.vbo)

	gl.BindVertexArray(s.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*4, 0)
	gl.EnableVertexAttribArray(0)

	gl.BindVertexArray(0)
}

// Update updates the sky state
func (s *Sky) Update(dt float32) {
	// Advance time of day
	s.TimeOfDay += (24.0 / s.DayLength) * dt
	if s.TimeOfDay >= 24.0 {
		s.TimeOfDay -= 24.0
	}
}

// GetDayNightFactor returns a factor from 0 (night) to 1 (day)
func (s *Sky) GetDayNightFactor() float32 {
	// Dawn: 5-7, Day: 7-17, Dusk: 17-19, Night: 19-5
	hour := s.TimeOfDay

	if hour >= 7 && hour < 17 {
		return 1.0
	} else if hour >= 5 && hour < 7 {
		// Dawn
		return (hour - 5) / 2.0
	} else if hour >= 17 && hour < 19 {
		// Dusk
		return 1.0 - (hour-17)/2.0
	}
	return 0.0
}

// GetSunDirection returns the sun direction vector
func (s *Sky) GetSunDirection() mgl32.Vec3 {
	// Sun angle based on time of day
	angle := (s.TimeOfDay - 6.0) / 12.0 * math.Pi // 6AM at horizon, noon at top

	return mgl32.Vec3{
		0,
		float32(-math.Sin(float64(angle))),
		float32(math.Cos(float64(angle))),
	}.Normalize()
}

// GetSkyColors returns the top and bottom sky colors
func (s *Sky) GetSkyColors() (top, bottom mgl32.Vec3) {
	factor := s.GetDayNightFactor()

	// Day colors
	dayTop := mgl32.Vec3{0.4, 0.7, 1.0}
	dayBottom := mgl32.Vec3{0.7, 0.85, 1.0}

	// Night colors
	nightTop := mgl32.Vec3{0.02, 0.02, 0.05}
	nightBottom := mgl32.Vec3{0.05, 0.05, 0.1}

	// Sunrise/sunset colors
	if s.TimeOfDay >= 5 && s.TimeOfDay < 7 {
		// Dawn
		dawnFactor := (s.TimeOfDay - 5) / 2.0
		if dawnFactor < 0.5 {
			// Early dawn - orange
			dayBottom = mgl32.Vec3{1.0, 0.5, 0.3}
		}
	} else if s.TimeOfDay >= 17 && s.TimeOfDay < 19 {
		// Dusk
		duskFactor := (s.TimeOfDay - 17) / 2.0
		if duskFactor < 0.5 {
			// Early dusk - orange/red
			dayBottom = mgl32.Vec3{1.0, 0.4, 0.2}
		}
	}

	top = lerpVec3(nightTop, dayTop, factor)
	bottom = lerpVec3(nightBottom, dayBottom, factor)

	return top, bottom
}

// GetFogColor returns the fog color
func (s *Sky) GetFogColor() mgl32.Vec3 {
	_, bottom := s.GetSkyColors()
	return bottom
}

// Render renders the sky
func (s *Sky) Render(view, projection mgl32.Mat4) {
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false)

	s.shader.Use()
	s.shader.SetMat4("view", view)
	s.shader.SetMat4("projection", projection)

	top, bottom := s.GetSkyColors()
	s.shader.SetVec3("skyColorTop", top)
	s.shader.SetVec3("skyColorBottom", bottom)
	s.shader.SetVec3("sunDir", s.GetSunDirection())
	s.shader.SetFloat("dayNightFactor", s.GetDayNightFactor())

	gl.BindVertexArray(s.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, 36)
	gl.BindVertexArray(0)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

// Cleanup releases resources
func (s *Sky) Cleanup() {
	gl.DeleteVertexArrays(1, &s.vao)
	gl.DeleteBuffers(1, &s.vbo)
	s.shader.Delete()
}

func lerpVec3(a, b mgl32.Vec3, t float32) mgl32.Vec3 {
	return mgl32.Vec3{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
	}
}
