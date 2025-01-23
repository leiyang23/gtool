package config

type Object struct {
	Name        string
	ShapeOption ShapeOption
}

type ShapeOption struct {
	Length int
	Width  int
	Height int
}

type Opt interface {
	ApplyTo(*Object)
}

type WithName string

func (w WithName) ApplyTo(obj *Object) {
	obj.Name = string(w)
}

type WithShapeOption struct {
	Length int
	Width  int
	Height int
}

func (w WithShapeOption) ApplyTo(obj *Object) {
	obj.ShapeOption = ShapeOption{
		w.Length,
		w.Width,
		w.Height,
	}
}

// defaultObject 返回 Object 的默认值
func defaultObject() *Object {
	return &Object{
		Name:        "default",
		ShapeOption: ShapeOption{Length: 1, Width: 1, Height: 1},
	}
}

// New 创建一个新的 Object 实例，并应用可选参数
func New(opts ...Opt) *Object {
	obj := defaultObject()

	for _, opt := range opts {
		opt.ApplyTo(obj)
	}

	return obj
}
