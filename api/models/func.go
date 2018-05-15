package models

import (
	"time"

	"github.com/go-openapi/strfmt"
)

// FuncWrapper makes jason purrty for a Func.
type FuncWrapper struct {
	Func *Func `json:"func"`
}

// Func contains information about a function configuration.
type Func struct {
	// ID is the generated resource id.
	ID string `json:"id" db:"id"`
	// Name is a user provided name for this func.
	Name string `json:"name" db:"name"`
	// Image is the fully qualified container registry address to execute.
	// examples: hub.docker.io/me/myfunc, me/myfunc, me/func:0.0.1
	Image string `json:"image" db:"image"`
	// Version is a user specified version for this func. It should be unique
	// (this is not enforced at the moment, however).
	Version string `json:"version" db:"version"`
	// ResourceConfig specifies resource constraints.
	ResourceConfig // embed
	// Config is the configuration passed to a function at execution time.
	Config Config `json:"config" db:"config"`
	// Annotations allow additional configuration of a function, these are not passed to the function.
	Annotations Annotations `json:"annotations,omitempty" db:"annotations"`
	// CreatedAt is the UTC timestamp when this function was created.
	CreatedAt strfmt.DateTime `json:"created_at,omitempty" db:"created_at"`
	// UpdatedAt is the UTC timestamp of the last time this func was modified.
	UpdatedAt strfmt.DateTime `json:"updated_at,omitempty" db:"updated_at"`

	// TODO wish to kill but not yet ?
	// Format is the container protocol the function will accept,
	// may be one of: json | http | cloudevent | default
	Format string `json:"format" db:"format"`
}

// ResourceConfig specified resource constraints imposed on a function execution.
type ResourceConfig struct {
	// Memory is the amount of memory allotted, in MB.
	Memory uint64 `json:"memory,omitempty" db:"memory"`
	// CPUs is the max usable CPU cores for this route. Value in MilliCPUs,
	// (eg. 500m) or as floating-point (eg. 0.5)
	// TODO this is a good chance to fix this. let's pick one?
	CPUs MilliCPUs `json:"cpus,omitempty" db:"cpus"`
	// Timeout is the max execution time for a function, in seconds.
	// TODO this should probably be milliseconds?
	Timeout int32 `json:"timeout,omitempty" db:"timeout"`
	// IdleTimeout is the
	// TODO this should probably be milliseconds
	IdleTimeout int32 `json:"idle_timeout,omitempty" db:"idle_timeout"`
}

// SetDefaults sets zeroed field to defaults.
func (f *Func) SetDefaults() {
	if f.Memory == 0 {
		f.Memory = DefaultMemory
	}

	if f.Format == "" {
		f.Format = FormatDefault
	}

	if f.Config == nil {
		// keeps the json from being nil
		f.Config = map[string]string{}
	}

	if f.Timeout == 0 {
		f.Timeout = DefaultTimeout
	}

	if f.IdleTimeout == 0 {
		f.IdleTimeout = DefaultIdleTimeout
	}

	if time.Time(f.CreatedAt).IsZero() {
		f.CreatedAt = strfmt.DateTime(time.Now())
	}

	if time.Time(f.UpdatedAt).IsZero() {
		f.UpdatedAt = strfmt.DateTime(time.Now())
	}
}

// Validate validates all field values, returning the first error, if any.
func (f *Func) Validate() error {
	if f.Image == "" {
		return ErrRoutesMissingImage
	}

	if f.Format != FormatDefault && f.Format != FormatHTTP && f.Format != FormatJSON && f.Format != FormatCloudEvent {
		return ErrRoutesInvalidFormat
	}

	if f.Timeout <= 0 || f.Timeout > MaxSyncTimeout {
		return ErrRoutesInvalidTimeout
	}

	if f.IdleTimeout <= 0 || f.IdleTimeout > MaxIdleTimeout {
		return ErrRoutesInvalidIdleTimeout
	}

	if f.Memory < 1 || f.Memory > RouteMaxMemory {
		return ErrRoutesInvalidMemory
	}

	return f.Annotations.Validate()
}

func (f *Func) Clone() *Func {
	clone := new(Func)
	*clone = *f // shallow copy

	// now deep copy the maps
	if f.Config != nil {
		clone.Config = make(Config, len(f.Config))
		for k, v := range f.Config {
			clone.Config[k] = v
		}
	}
	if f.Annotations != nil {
		clone.Annotations = make(Annotations, len(f.Annotations))
		for k, v := range f.Annotations {
			// TODO technically, we need to deep copy the bytes
			clone.Annotations[k] = v
		}
	}
	return clone
}

func (f1 *Func) Equals(f2 *Func) bool {
	// start off equal, check equivalence of each field.
	// the RHS of && won't eval if eq==false so config/headers checking is lazy

	eq := true
	eq = eq && f1.Image == f2.Image
	eq = eq && f1.Memory == f2.Memory
	eq = eq && f1.CPUs == f2.CPUs
	eq = eq && f1.Format == f2.Format
	eq = eq && f1.Timeout == f2.Timeout
	eq = eq && f1.IdleTimeout == f2.IdleTimeout
	eq = eq && f1.Config.Equals(f2.Config)
	eq = eq && f1.Annotations.Equals(f2.Annotations)
	// NOTE: datastore tests are not very fun to write with timestamp checks,
	// and these are not values the user may set so we kind of don't care.
	//eq = eq && time.Time(f1.CreatedAt).Equal(time.Time(f2.CreatedAt))
	//eq = eq && time.Time(f2.UpdatedAt).Equal(time.Time(f2.UpdatedAt))
	return eq
}

// Update updates fields in f with non-zero field values from new, and sets
// updated_at if any of the fields change. 0-length slice Header values, and
// empty-string Config values trigger removal of map entry.
func (f *Func) Update(patch *Route) {
	original := f.Clone()

	if patch.Image != "" {
		f.Image = patch.Image
	}
	if patch.Memory != 0 {
		f.Memory = patch.Memory
	}
	if patch.CPUs != 0 {
		f.CPUs = patch.CPUs
	}
	if patch.Timeout != 0 {
		f.Timeout = patch.Timeout
	}
	if patch.IdleTimeout != 0 {
		f.IdleTimeout = patch.IdleTimeout
	}
	if patch.Format != "" {
		f.Format = patch.Format
	}
	if patch.Config != nil {
		if f.Config == nil {
			f.Config = make(Config)
		}
		for k, v := range patch.Config {
			if v == "" {
				delete(f.Config, k)
			} else {
				f.Config[k] = v
			}
		}
	}

	f.Annotations = f.Annotations.MergeChange(patch.Annotations)

	if !f.Equals(original) {
		f.UpdatedAt = strfmt.DateTime(time.Now())
	}
}

type FuncFilter struct {
	Image string // this is exact match

	Cursor  string
	PerPage int
}