(function(abot) {
abot.ChatboxItem = {}
abot.ChatboxItem.view = function(_, props) {
	var c = props.abotSent ? "chat-abot" : "chat-user"
	var u = props.abotSent ? "abot" : props.Username
	return m("li", { class: c }, [
		m(".messages", [
			m("p", props.Sentence),
			m("time", {
				datetime: props.CreatedAt
			}, u + " • " + abot.prettyDate(props.CreatedAt))
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
