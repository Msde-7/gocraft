package utils

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Clamp restricts a value to a range
func Clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampInt restricts an integer value to a range
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Lerp performs linear interpolation
func Lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

// SmoothStep performs smooth interpolation
func SmoothStep(edge0, edge1, x float32) float32 {
	t := Clamp((x-edge0)/(edge1-edge0), 0.0, 1.0)
	return t * t * (3.0 - 2.0*t)
}

// Floor returns the floor of a float32 as int
func Floor(x float32) int {
	if x >= 0 {
		return int(x)
	}
	return int(x) - 1
}

// Mod performs modulo operation that handles negative numbers correctly
func Mod(a, b int) int {
	return ((a % b) + b) % b
}

// Vec3i represents an integer 3D vector
type Vec3i struct {
	X, Y, Z int
}

// Add adds two Vec3i
func (v Vec3i) Add(other Vec3i) Vec3i {
	return Vec3i{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// ToFloat converts to mgl32.Vec3
func (v Vec3i) ToFloat() mgl32.Vec3 {
	return mgl32.Vec3{float32(v.X), float32(v.Y), float32(v.Z)}
}

// AABB represents an axis-aligned bounding box
type AABB struct {
	Min, Max mgl32.Vec3
}

// NewAABB creates a new AABB
func NewAABB(min, max mgl32.Vec3) AABB {
	return AABB{Min: min, Max: max}
}

// Intersects checks if two AABBs intersect
func (a AABB) Intersects(b AABB) bool {
	return a.Min[0] <= b.Max[0] && a.Max[0] >= b.Min[0] &&
		a.Min[1] <= b.Max[1] && a.Max[1] >= b.Min[1] &&
		a.Min[2] <= b.Max[2] && a.Max[2] >= b.Min[2]
}

// Contains checks if a point is inside the AABB
func (a AABB) Contains(point mgl32.Vec3) bool {
	return point[0] >= a.Min[0] && point[0] <= a.Max[0] &&
		point[1] >= a.Min[1] && point[1] <= a.Max[1] &&
		point[2] >= a.Min[2] && point[2] <= a.Max[2]
}

// Expand expands the AABB by a given amount
func (a AABB) Expand(amount mgl32.Vec3) AABB {
	return AABB{
		Min: a.Min.Sub(amount),
		Max: a.Max.Add(amount),
	}
}

// Translate moves the AABB by a given amount
func (a AABB) Translate(offset mgl32.Vec3) AABB {
	return AABB{
		Min: a.Min.Add(offset),
		Max: a.Max.Add(offset),
	}
}

// Frustum represents a view frustum for culling
type Frustum struct {
	planes [6]mgl32.Vec4
}

// ExtractFrustum extracts frustum planes from a view-projection matrix
func ExtractFrustum(vp mgl32.Mat4) Frustum {
	var f Frustum

	// Left plane
	f.planes[0] = mgl32.Vec4{
		vp[3] + vp[0],
		vp[7] + vp[4],
		vp[11] + vp[8],
		vp[15] + vp[12],
	}

	// Right plane
	f.planes[1] = mgl32.Vec4{
		vp[3] - vp[0],
		vp[7] - vp[4],
		vp[11] - vp[8],
		vp[15] - vp[12],
	}

	// Bottom plane
	f.planes[2] = mgl32.Vec4{
		vp[3] + vp[1],
		vp[7] + vp[5],
		vp[11] + vp[9],
		vp[15] + vp[13],
	}

	// Top plane
	f.planes[3] = mgl32.Vec4{
		vp[3] - vp[1],
		vp[7] - vp[5],
		vp[11] - vp[9],
		vp[15] - vp[13],
	}

	// Near plane
	f.planes[4] = mgl32.Vec4{
		vp[3] + vp[2],
		vp[7] + vp[6],
		vp[11] + vp[10],
		vp[15] + vp[14],
	}

	// Far plane
	f.planes[5] = mgl32.Vec4{
		vp[3] - vp[2],
		vp[7] - vp[6],
		vp[11] - vp[10],
		vp[15] - vp[14],
	}

	// Normalize planes
	for i := range f.planes {
		length := float32(math.Sqrt(float64(
			f.planes[i][0]*f.planes[i][0] +
				f.planes[i][1]*f.planes[i][1] +
				f.planes[i][2]*f.planes[i][2])))
		if length > 0 {
			f.planes[i][0] /= length
			f.planes[i][1] /= length
			f.planes[i][2] /= length
			f.planes[i][3] /= length
		}
	}

	return f
}

// ContainsAABB checks if an AABB is inside or intersects the frustum
func (f Frustum) ContainsAABB(aabb AABB) bool {
	for _, plane := range f.planes {
		// Find the positive vertex (the one furthest along the plane normal)
		px := aabb.Min[0]
		if plane[0] >= 0 {
			px = aabb.Max[0]
		}
		py := aabb.Min[1]
		if plane[1] >= 0 {
			py = aabb.Max[1]
		}
		pz := aabb.Min[2]
		if plane[2] >= 0 {
			pz = aabb.Max[2]
		}

		// If positive vertex is outside, the AABB is completely outside
		if plane[0]*px+plane[1]*py+plane[2]*pz+plane[3] < 0 {
			return false
		}
	}
	return true
}

// Ray represents a ray for raycasting
type Ray struct {
	Origin    mgl32.Vec3
	Direction mgl32.Vec3
}

// NewRay creates a new ray
func NewRay(origin, direction mgl32.Vec3) Ray {
	return Ray{Origin: origin, Direction: direction.Normalize()}
}

// At returns the point at distance t along the ray
func (r Ray) At(t float32) mgl32.Vec3 {
	return r.Origin.Add(r.Direction.Mul(t))
}

// RaycastResult holds the result of a raycast
type RaycastResult struct {
	Hit      bool
	Position mgl32.Vec3
	BlockPos Vec3i
	Normal   mgl32.Vec3
	Face     int
	Distance float32
}
