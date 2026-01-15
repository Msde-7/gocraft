package render

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Shader represents an OpenGL shader program
type Shader struct {
	ID uint32
}

// NewShader compiles and links a shader program from vertex and fragment source
func NewShader(vertexSource, fragmentSource string) (*Shader, error) {
	vertexShader, err := compileShader(vertexSource, gl.VERTEX_SHADER)
	if err != nil {
		return nil, fmt.Errorf("vertex shader: %w", err)
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := compileShader(fragmentSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, fmt.Errorf("fragment shader: %w", err)
	}
	defer gl.DeleteShader(fragmentShader)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return nil, fmt.Errorf("link error: %s", log)
	}

	return &Shader{ID: program}, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csource, free := gl.Strs(source + "\x00")
	defer free()
	gl.ShaderSource(shader, 1, csource, nil)
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("compile error: %s", log)
	}

	return shader, nil
}

// Use activates the shader program
func (s *Shader) Use() {
	gl.UseProgram(s.ID)
}

// Delete cleans up the shader program
func (s *Shader) Delete() {
	gl.DeleteProgram(s.ID)
}

// SetInt sets an integer uniform
func (s *Shader) SetInt(name string, value int32) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.Uniform1i(loc, value)
}

// SetFloat sets a float uniform
func (s *Shader) SetFloat(name string, value float32) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.Uniform1f(loc, value)
}

// SetVec2 sets a vec2 uniform
func (s *Shader) SetVec2(name string, value mgl32.Vec2) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.Uniform2fv(loc, 1, &value[0])
}

// SetVec3 sets a vec3 uniform
func (s *Shader) SetVec3(name string, value mgl32.Vec3) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.Uniform3fv(loc, 1, &value[0])
}

// SetVec4 sets a vec4 uniform
func (s *Shader) SetVec4(name string, value mgl32.Vec4) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.Uniform4fv(loc, 1, &value[0])
}

// SetMat4 sets a mat4 uniform
func (s *Shader) SetMat4(name string, value mgl32.Mat4) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	gl.UniformMatrix4fv(loc, 1, false, &value[0])
}

// BlockShaderVertex is the vertex shader for blocks
const BlockShaderVertex = `
#version 410 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoord;
layout (location = 2) in vec3 aNormal;
layout (location = 3) in float aAO;
layout (location = 4) in float aLight;

out vec2 TexCoord;
out vec3 Normal;
out vec3 FragPos;
out float AO;
out float Light;
out float FogFactor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;
uniform vec3 cameraPos;
uniform float fogStart;
uniform float fogEnd;

void main() {
    vec4 worldPos = model * vec4(aPos, 1.0);
    FragPos = worldPos.xyz;
    gl_Position = projection * view * worldPos;

    TexCoord = aTexCoord;
    Normal = mat3(transpose(inverse(model))) * aNormal;
    AO = aAO;
    Light = aLight;

    // Calculate fog
    float dist = length(cameraPos - FragPos);
    FogFactor = clamp((fogEnd - dist) / (fogEnd - fogStart), 0.0, 1.0);
}
`

// BlockShaderFragment is the fragment shader for blocks
const BlockShaderFragment = `
#version 410 core

in vec2 TexCoord;
in vec3 Normal;
in vec3 FragPos;
in float AO;
in float Light;
in float FogFactor;

out vec4 FragColor;

uniform sampler2D textureSampler;
uniform vec3 lightDir;
uniform vec3 lightColor;
uniform vec3 ambientColor;
uniform vec3 fogColor;
uniform float dayNightFactor;

void main() {
    vec4 texColor = texture(textureSampler, TexCoord);
    if (texColor.a < 0.5) {
        discard;
    }

    // Normalize the normal
    vec3 norm = normalize(Normal);

    // Diffuse lighting
    float diff = max(dot(norm, -lightDir), 0.0);
    vec3 diffuse = diff * lightColor * dayNightFactor;

    // Combine lighting
    vec3 ambient = ambientColor * (0.3 + 0.7 * dayNightFactor);
    vec3 lighting = ambient + diffuse;

    // Apply ambient occlusion
    lighting *= AO;

    // Apply block light
    lighting = max(lighting, vec3(Light * 0.9));

    vec3 result = texColor.rgb * lighting;

    // Apply fog
    result = mix(fogColor, result, FogFactor);

    FragColor = vec4(result, texColor.a);
}
`

// SkyShaderVertex is the vertex shader for the sky
const SkyShaderVertex = `
#version 410 core

layout (location = 0) in vec3 aPos;

out vec3 TexCoords;

uniform mat4 view;
uniform mat4 projection;

void main() {
    TexCoords = aPos;
    vec4 pos = projection * mat4(mat3(view)) * vec4(aPos, 1.0);
    gl_Position = pos.xyww;
}
`

// SkyShaderFragment is the fragment shader for the sky
const SkyShaderFragment = `
#version 410 core

in vec3 TexCoords;
out vec4 FragColor;

uniform vec3 skyColorTop;
uniform vec3 skyColorBottom;
uniform vec3 sunDir;
uniform float dayNightFactor;

void main() {
    // Gradient sky based on height
    float height = normalize(TexCoords).y;
    height = height * 0.5 + 0.5; // Map from [-1,1] to [0,1]

    vec3 skyColor = mix(skyColorBottom, skyColorTop, height);

    // Add sun glow
    vec3 dir = normalize(TexCoords);
    float sunDot = max(dot(dir, -sunDir), 0.0);
    float sunGlow = pow(sunDot, 128.0);
    float sunDisc = smoothstep(0.997, 0.999, sunDot);

    vec3 sunColor = vec3(1.0, 0.95, 0.8);
    skyColor += sunGlow * sunColor * 0.5 * dayNightFactor;
    skyColor += sunDisc * sunColor * dayNightFactor;

    // Stars at night
    float nightFactor = 1.0 - dayNightFactor;
    if (nightFactor > 0.3 && height > 0.3) {
        // Simple star pattern using noise-like effect
        vec3 starPos = floor(TexCoords * 100.0);
        float starHash = fract(sin(dot(starPos, vec3(12.9898, 78.233, 45.164))) * 43758.5453);
        if (starHash > 0.99) {
            float twinkle = 0.5 + 0.5 * sin(starHash * 1000.0);
            skyColor += vec3(twinkle) * nightFactor * 0.8;
        }
    }

    FragColor = vec4(skyColor, 1.0);
}
`

// UIShaderVertex is the vertex shader for UI elements
const UIShaderVertex = `
#version 410 core

layout (location = 0) in vec2 aPos;
layout (location = 1) in vec2 aTexCoord;

out vec2 TexCoord;

uniform mat4 projection;

void main() {
    gl_Position = projection * vec4(aPos, 0.0, 1.0);
    TexCoord = aTexCoord;
}
`

// UIShaderFragment is the fragment shader for UI elements
const UIShaderFragment = `
#version 410 core

in vec2 TexCoord;
out vec4 FragColor;

uniform sampler2D textureSampler;
uniform vec4 color;
uniform bool useTexture;

void main() {
    if (useTexture) {
        FragColor = texture(textureSampler, TexCoord) * color;
    } else {
        FragColor = color;
    }
}
`

// WireframeShaderVertex is the vertex shader for wireframe rendering
const WireframeShaderVertex = `
#version 410 core

layout (location = 0) in vec3 aPos;

uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;

void main() {
    gl_Position = projection * view * model * vec4(aPos, 1.0);
}
`

// WireframeShaderFragment is the fragment shader for wireframe rendering
const WireframeShaderFragment = `
#version 410 core

out vec4 FragColor;
uniform vec4 color;

void main() {
    FragColor = color;
}
`
