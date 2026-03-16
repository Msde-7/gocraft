package render

import (
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// CloudShaderVertex is the vertex shader for clouds
const CloudShaderVertex = `
#version 410 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoord;

out vec2 TexCoord;
out float FogFactor;

uniform mat4 view;
uniform mat4 projection;
uniform vec3 cameraPos;
uniform float fogEnd;

void main() {
    gl_Position = projection * view * vec4(aPos, 1.0);
    TexCoord = aTexCoord;

    float dist = length(cameraPos.xz - aPos.xz);
    FogFactor = clamp(1.0 - (dist / fogEnd), 0.0, 1.0);
}
`

// CloudShaderFragment is the fragment shader for clouds
const CloudShaderFragment = `
#version 410 core

in vec2 TexCoord;
in float FogFactor;

out vec4 FragColor;

uniform float dayNightFactor;
uniform vec3 fogColor;
uniform float time;

// Simple hash for cloud noise
float hash(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}

// Value noise
float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);

    float a = hash(i);
    float b = hash(i + vec2(1.0, 0.0));
    float c = hash(i + vec2(0.0, 1.0));
    float d = hash(i + vec2(1.0, 1.0));

    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

// FBM (fractal Brownian motion) for cloud shape
float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    for (int i = 0; i < 4; i++) {
        value += amplitude * noise(p);
        p *= 2.0;
        amplitude *= 0.5;
    }
    return value;
}

void main() {
    // Animate clouds slowly
    vec2 uv = TexCoord + vec2(time * 0.002, time * 0.001);

    float cloudDensity = fbm(uv * 3.0);

    // Threshold for cloud shape
    float cloudAlpha = smoothstep(0.35, 0.55, cloudDensity);

    if (cloudAlpha < 0.01) {
        discard;
    }

    // Cloud color based on day/night
    vec3 cloudColor = mix(vec3(0.15, 0.15, 0.2), vec3(1.0, 1.0, 1.0), dayNightFactor);

    // Fade at edges for soft look
    cloudAlpha *= FogFactor * 0.7;

    // Mix with fog at distance
    cloudColor = mix(fogColor, cloudColor, FogFactor);

    FragColor = vec4(cloudColor, cloudAlpha);
}
`

// Clouds renders a cloud layer
type Clouds struct {
	shader    *Shader
	vao       uint32
	vbo       uint32
	ebo       uint32
	indexCount int32
	cloudHeight float32
	gridSize    float32
	time      float32
}

// NewClouds creates a new cloud renderer
func NewClouds() (*Clouds, error) {
	shader, err := NewShader(CloudShaderVertex, CloudShaderFragment)
	if err != nil {
		return nil, err
	}

	c := &Clouds{
		shader:      shader,
		cloudHeight: 128.0,
		gridSize:    256.0,
	}

	c.createMesh()
	return c, nil
}

func (c *Clouds) createMesh() {
	halfSize := c.gridSize

	// Simple quad centered on player (will be repositioned each frame)
	vertices := []float32{
		// x, y, z, u, v
		-halfSize, c.cloudHeight, -halfSize, 0, 0,
		halfSize, c.cloudHeight, -halfSize, 1, 0,
		halfSize, c.cloudHeight, halfSize, 1, 1,
		-halfSize, c.cloudHeight, halfSize, 0, 1,
	}

	indices := []uint32{
		0, 1, 2,
		2, 3, 0,
	}

	gl.GenVertexArrays(1, &c.vao)
	gl.GenBuffers(1, &c.vbo)
	gl.GenBuffers(1, &c.ebo)

	gl.BindVertexArray(c.vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, c.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	// Position
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 5*4, 0)
	gl.EnableVertexAttribArray(0)

	// TexCoord
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 5*4, 3*4)
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	c.indexCount = int32(len(indices))
}

// Update advances cloud animation
func (c *Clouds) Update(dt float32) {
	c.time += dt
}

// Render draws the clouds
func (c *Clouds) Render(view, projection mgl32.Mat4, cameraPos mgl32.Vec3, fogColor mgl32.Vec3, fogEnd float32, dayNightFactor float32) {
	// Reposition cloud quad centered on player
	halfSize := c.gridSize
	px := float32(math.Floor(float64(cameraPos[0])))
	pz := float32(math.Floor(float64(cameraPos[2])))

	vertices := []float32{
		px - halfSize, c.cloudHeight, pz - halfSize, 0, 0,
		px + halfSize, c.cloudHeight, pz - halfSize, 1, 0,
		px + halfSize, c.cloudHeight, pz + halfSize, 1, 1,
		px - halfSize, c.cloudHeight, pz + halfSize, 0, 1,
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))

	gl.DepthMask(false) // Don't write to depth buffer
	gl.Disable(gl.CULL_FACE)

	c.shader.Use()
	c.shader.SetMat4("view", view)
	c.shader.SetMat4("projection", projection)
	c.shader.SetVec3("cameraPos", cameraPos)
	c.shader.SetVec3("fogColor", fogColor)
	c.shader.SetFloat("fogEnd", fogEnd)
	c.shader.SetFloat("dayNightFactor", dayNightFactor)
	c.shader.SetFloat("time", c.time)

	gl.BindVertexArray(c.vao)
	gl.DrawElements(gl.TRIANGLES, c.indexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)

	gl.DepthMask(true)
	gl.Enable(gl.CULL_FACE)
}

// Cleanup releases resources
func (c *Clouds) Cleanup() {
	gl.DeleteVertexArrays(1, &c.vao)
	gl.DeleteBuffers(1, &c.vbo)
	gl.DeleteBuffers(1, &c.ebo)
	c.shader.Delete()
}
