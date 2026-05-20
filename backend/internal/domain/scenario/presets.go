package scenario

import "encoding/json"

func PresetCommits() json.RawMessage {
	return json.RawMessage(`{"nodes":[{"id":"t1","type":"trigger.cron","parameters":{"hour":18,"minute":30,"weekdays":[1,2,3,4,5]}},{"id":"a1","type":"action.telegram.message","parameters":{"text":"Пора коммитить! 🐾"}},{"id":"a2","type":"action.telegram.cat_photo","parameters":{}}],"edges":[{"source":"t1","target":"a1"},{"source":"a1","target":"a2"}]}`)
}

func PresetMeet() json.RawMessage {
	return json.RawMessage(`{"nodes":[{"id":"t1","type":"trigger.cron","parameters":{"hour":10,"minute":15,"weekdays":[1,3,5]}},{"id":"a1","type":"action.telegram.message","parameters":{"text":"Созвон через 15 минут! 🐾 {{meet_link}}"}}],"edges":[{"source":"t1","target":"a1"}]}`)
}

func PresetReport() json.RawMessage {
	return json.RawMessage(`{"nodes":[{"id":"t1","type":"trigger.cron","parameters":{"hour":18,"minute":35,"weekdays":[1,2,3,4,5]}},{"id":"a1","type":"action.vcs.commits_report","parameters":{}}],"edges":[{"source":"t1","target":"a1"}]}`)
}
