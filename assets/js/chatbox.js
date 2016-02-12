(function(ava) {
ava.Chatbox = {}
ava.Chatbox.controller = function(_, pctrl) {
	var ctrl = this
	ctrl.handleSend = function(ev) {
		if (ev.keyCode === 13 /* enter */ && !ev.shiftKey) {
			ev.preventDefault()
			var sentence = ev.srcElement.value
			// TODO provide some security here ensuring the user has an open
			// needsTraining message, preventing the trainer from swapping the
			// userid to send messages to new users. OR create a tmp uuid to ID the
			// user that's cycled every so often
			ctrl.send(m.route.param("uid"), sentence).then(function() {
				// TODO there's a better way to do this...
				//ctrl.vm.showAvaMsg(sentence)
				pctrl.loadConversation()
			}, function(err) {
				// TODO display error to the user
				console.error(err)
				ev.srcElement.value = sentence
			})
			ev.srcElement.value = ""
			return
		}
	}
	ctrl.scrollToBottom = function(ev) {
		ev.scrollTop = ev.scrollHeight
	}
	ctrl.send = function(uid, sentence) {
		if (ctrl.props.Contact().length === 0) {
			// Send to the user
			return m.request({
				method: "POST",
				url: "/api/conversations.json",
				data: {
					UserID: parseInt(uid),
					Sentence: sentence,
					Contact: ctrl.props.Contact(),
					ContactMethod: ctrl.props.ContactMethod(),
				}
			})
		}
		// Else send to a contact
		return m.request({
			method: "POST",
			url: "/api/contacts/conversations.json",
			data: {
				UserID: parseInt(uid),
				Sentence: sentence,
				Contact: ctrl.props.Contact(),
				ContactMethod: ctrl.props.ContactMethod(),
			}
		})
	}
	ctrl.props = {
		Contact: m.prop(""),
		ContactMethod: m.prop(""), // Email or Phone
	}
}
ava.Chatbox.view = function(ctrl, props) {
	if (props == null) {
		props = {
			Username: m.prop(""),
			Messages: m.prop([]),
			UsernameDisabled: "",
		}
	}
	return m("div.chat-container", [
		m.component(ava.Searchbox, props, ctrl),
		m("ol#chat-box.chat-box", {
			config: ctrl.scrollToBottom,
		}, props.Messages().map(function(c) {
			var d = {
				Username: props.Username(),
				Sentence: c.Sentence,
				AvaSent: c.AvaSent,
				CreatedAt: c.CreatedAt,
			}
			return m.component(ava.ChatboxItem, d)
		})),
		m("textarea.chat-textarea[rows=4][placeholder=Your message]", {
			onkeydown: ctrl.handleSend,
		})
	])
}
})(!window.ava ? window.ava={} : window.ava);
