package constants

// 错题掌握状态枚举：unmastered（未掌握）-> learning（巩固中）-> mastered（已掌握）。
// 同时出现在 model.Mistake.Mastery、DTO、service 复习排期、handler 校验、
// util/formatters.go、前端 constants/index.ts、错误码、日志模板中。
const (
	MasteryUnmastered = "unmastered"
	MasteryLearning   = "learning"
	MasteryMastered   = "mastered"
)

// MasteryReviewIntervalDays 按掌握状态的默认复习间隔（天）：掌握越好间隔越长。
var MasteryReviewIntervalDays = map[string]int{
	MasteryUnmastered: 1,
	MasteryLearning:   3,
	MasteryMastered:   7,
}

// ValidMastery 校验掌握状态是否合法。
func ValidMastery(m string) bool {
	_, ok := MasteryReviewIntervalDays[m]
	return ok
}
