package skills

import "sort"

var skillRegistry *SkillRegistry

type SkillRegistry struct {
	skills map[string]SkillDefinition
}

func NewSkillRegistry() *SkillRegistry {
	if skillRegistry != nil {
		return skillRegistry
	}
	
	skillRegistry = &SkillRegistry{
		skills: make(map[string]SkillDefinition),
	}
	return skillRegistry
}

func (sr *SkillRegistry) Register(skill SkillDefinition) {
	sr.skills[skill.Name] = skill
}

func (sr *SkillRegistry) SkillDefinition(name string) SkillDefinition {
	return sr.skills[name]
}

func (sr *SkillRegistry) GetSkills() []SkillDefinition {
	result := make([]SkillDefinition, 0, len(sr.skills))
	for _, skill := range sr.skills {
		result = append(result, skill)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
