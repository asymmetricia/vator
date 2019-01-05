package models

var encourageToasts = []string{
	"your {{days}}-day average is {{direction}} {{delta}}{{unit}} to {{final}}{{unit}}- small roadbump, but that's ok!",
	"hi.. {{direction}} a little, I'm afraid. Just {{delta}}{{unit}}. Your {{days}}-day average is now {{final}}{{unit}}, but one good day can turn that around",
}

var notEnoughData = []string{
	"welcome back! I don't have enough measurements to calculate trends, but maybe if I see you tomorrow...",
	"good to see you! I need another day or two of measurements before I can see what's going on.",
	"hi! thanks for stepping on the scale today. I don't have an update right now, but I might have something tomorrow.",
}

var toasts = []string{
	"nice! your {{days}} day average is {{direction}} by {{delta}}{{unit}} to {{final}}{{unit}}",
	"cool, that brings your {{days}}-day average {{direction}} {{delta}}{{unit}} to {{final}}{{unit}}",
}
