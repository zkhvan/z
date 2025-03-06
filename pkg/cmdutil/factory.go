package cmdutil

import (
	"time"

	"github.com/zkhvan/z/pkg/iolib"
)

type Factory struct {
	AppVersion     string
	ExecutableName string

	PluginHandler PluginHandler
	IOStreams     *iolib.IOStreams
	Config        Config
}

type Config interface {
	Bool(path string) bool
	BoolMap(path string) map[string]bool
	Bools(path string) []bool
	Bytes(path string) []byte
	Duration(path string) time.Duration
	Float64(path string) float64
	Float64Map(path string) map[string]float64
	Float64s(path string) []float64
	Int(path string) int
	Int64(path string) int64
	Int64Map(path string) map[string]int64
	Int64s(path string) []int64
	IntMap(path string) map[string]int
	Ints(path string) []int
	String(path string) string
	StringMap(path string) map[string]string
	Strings(path string) []string
	StringsMap(path string) map[string][]string
	Time(path, layout string) time.Time

	Get(path string) any
	List() string
	Unmarshal(path string, v any) error
}
