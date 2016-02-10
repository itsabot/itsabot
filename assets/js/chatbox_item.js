(function(ava) {
ava.ChatboxItem = {}
ava.ChatboxItem.view = function(_, props) {
	var c = props.AvaSent ? "chat-ava" : "chat-user"
	var u = props.AvaSent ? "Ava" : props.Username
	return m("li.c", [
		m(".messages", [
			m("p", props.Sentence),
			m("time", {
				datetime: props.CreatedAt
			}, u + " • " + ava.prettyDate(props.CreatedAt))
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
