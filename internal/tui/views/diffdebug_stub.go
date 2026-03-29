//go:build !debug

package views

// DiffDebugModel is a no-op stub when not building with -tags debug.
type DiffDebugModel struct{}

func NewDiffDebugModel() DiffDebugModel    { return DiffDebugModel{} }
func (m *DiffDebugModel) SetSize(w, h int) {}
func (m *DiffDebugModel) ScrollUp()        {}
func (m *DiffDebugModel) ScrollDown()      {}
func (m *DiffDebugModel) View() string     { return "Build with -tags debug to enable diff preview." }
