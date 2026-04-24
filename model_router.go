package agent

type ModelLevel string

const (
	LevelFast   ModelLevel = "fast"   // Q2
	LevelNormal ModelLevel = "normal" // Q4
	LevelBest   ModelLevel = "best"   // Q8
)

type ModelRouter struct {
	mapping      map[ModelLevel]string
	defaultModel string
}

func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		mapping: map[ModelLevel]string{
			LevelFast:   "deepseek-q2",
			LevelNormal: "deepseek-q4",
			LevelBest:   "deepseek-q8",
		},
		defaultModel: "deepseek-q4",
	}
}

func (r *ModelRouter) Set(level ModelLevel, model string) {
	if r == nil {
		return
	}
	if r.mapping == nil {
		r.mapping = map[ModelLevel]string{}
	}
	r.mapping[level] = model
}

func (r *ModelRouter) SetDefault(model string) {
	if r == nil {
		return
	}
	r.defaultModel = model
}

func (r *ModelRouter) Get(level ModelLevel) string {
	if r == nil {
		return ""
	}
	if v, ok := r.mapping[level]; ok && v != "" {
		return v
	}
	if r.defaultModel != "" {
		return r.defaultModel
	}
	return r.mapping[LevelNormal]
}

