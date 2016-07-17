(function(abot) {
abot.ResponsePanelConversation = {}
abot.ResponsePanelConversation.controller = function() {
	var ctrl = this
	ctrl.scroll = function() {
		var el = document.getElementById("conversation")
		if (ctrl.props.offset() > 0) {
			el.scrollTop = 0
		} else {
			el.scrollTop = 1000000
		}
	}
	ctrl.focus = function(el) {
		el.focus()
	}
	ctrl.resolve = function() {
		abot.request({
			url: "/api/admin/conversations.json",
			method: "PATCH",
			data: {
				MessageID: ctrl.props.conversationID(),
				UserID: parseInt(m.route.param("uid")),
				FlexID: m.route.param("fid"),
				FlexIDType: parseInt(m.route.param("fidt")),
			}
		}).then(function() {
			m.route("/response_panel", null, true)
		}, function(err) {
			ctrl.props.error(err.Msg)
		})
	}
	ctrl.checkSubmit = function(ev) {
		if (!ctrl.props.helpShown()) {
			document.getElementById("help").classList.remove("hidden")
			ctrl.props.helpShown(true)
		}
		if (ev.keyCode !== 13 || ev.keyCode === 13 && ev.shiftKey) {
			return
		}
		ev.preventDefault()
		var val = ev.target.value
		if (val.length <= 1) {
			return
		}
		ev.target.value = ""
		var msg = {
			Sentence: val,
			AbotSent: true,
			CreatedAt: Date.now(),
		}
		ctrl.props.messages().push(msg)
		document.getElementById("help").classList.add("hidden")
		abot.request({
			url: "/api/admins/send_message.json",
			method: "POST",
			data: {
				UserID: parseInt(m.route.param("uid")),
				FlexID: m.route.param("fid"),
				FlexIDType: parseInt(m.route.param("fidt")),
				Name: ctrl.props.name(),
				Sentence: val,
			},
		}).then(null, function(err) {
			ev.target.value = val
			ctrl.props.error(err.Msg)
			ctrl.props.messages().pop()
		})
	}
	ctrl.loadMoreMsgs = function(ev) {
		var off = ctrl.props.offset()
		ctrl.props.offset(off + 30)
		abot.request({
			url: url + ctrl.props.offset(),
			method: "GET",
		}).then(function(resp) {
			if (resp.Messages.length < 30) {
				ev.target.classList.add("hidden")
			}
			for (var i = 0; i < resp.Messages.length; ++i) {
				ctrl.props.messages().unshift(resp.Messages[i])
			}
		}, function(err) {
			ctrl.props.error(err.Msg)
		})
	}
	ctrl.props = {
		conversationID: m.prop(0),
		messages: m.prop([]),
		flexIDs: m.prop([]),
		name: m.prop(""),
		lastSeen: m.prop(""),
		offset: m.prop(0),
		helpShown: m.prop(false),
		success: m.prop(""),
		error: m.prop(""),
	}
	var url = "/api/admin/conversations/" + m.route.param("uid") + "/" +
		m.route.param("fid") + "/" + m.route.param("fidt") + "/"
	abot.request({
		url: url + ctrl.props.offset(),
		method: "GET",
	}).then(function(resp) {
		ctrl.props.messages(resp.Messages)
		ctrl.props.name(resp.Name)
		ctrl.props.flexIDs(resp.FlexIDs)
		var msgs = ctrl.props.messages()
		if (msgs.length > 0) {
			var date = abot.prettyDate(msgs[0].CreatedAt)
			if (date != null) {
				ctrl.props.lastSeen(date)
			}
		}
	}, function(err) {
		ctrl.props.error(err.Msg)
	})
}
abot.ResponsePanelConversation.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 2 }),
			m(".main.sidebar-right", [
				m(".topbar", [
					m(".topbar-inline", "Response Panel"),
						m(".topbar-right", [
							m("a[href=#/]", {
								onclick: ctrl.resolve
							}, "Mark as solved"),
						]),
				]),
				m(".content-container", [
					m(".content", [
						function() {
							if (ctrl.props.error().length > 0) {
								return m(".alert.alert-danger.alert-margin", ctrl.props.error())
							}
							if (ctrl.props.success().length > 0) {
								return m(".alert.alert-success.alert-margin", ctrl.props.success())
							}
						}(),
						m("h3.top-el", "Conversation"),
						m(".chatbox", [
							m("#conversation.sheet-content-container", { config: ctrl.scroll }, [
								m(".conversation-parts-container", [
									m(".conversation-parts", [
										function() {
											if (ctrl.props.messages().length < 30) {
												return
											}
											return m(".conversation-parts-header", {
												onclick: ctrl.loadMoreMsgs,
											}, [
												m("label", "Load more messages"),
											])
										}(),
										ctrl.props.messages().map(function(msg) {
											var c = msg.AbotSent ? "message-by-user" : "message-by-abot"
											return m("div", { "class": c + " message" }, [
												m(".message-body-container", [
													m(".message-body .embed-body", [
														m("p", msg.Sentence),
														m(".comment-caret"),	
													]),
												]),
											])
										}),
										m(".message.last"),
									]),
								]),
							]),
							m(".composer", [
								m(".composer-textarea-container", [
									m(".composer-textarea", [
										m("textarea", {
											placeholder: "Write a reply...",
											onkeydown: ctrl.checkSubmit,
											config: ctrl.focus,
										}),
										m("label.composer-help.hidden#help", "Press enter to send."),
									]),
								]),
							]),	
						]),
					]),
				]),
			]),
			m(".sidebar.sidebar-right", [
				m("h5.top-el", "User Info"),
				m(".sidebar-content", [
					function() {
						if (ctrl.props.name().length === 0) {
							return
						}
						return [
							m("img[src=/public/images/clock.svg]"),
							m("b", "Name: "),
							ctrl.props.name(),
						]
					}(),
					function() {
						if (ctrl.props.lastSeen().length === 0) {
							return
						}
						return [
							m("img[src=/public/images/clock.svg]"),
							m("b", "Last seen: "),
							ctrl.props.lastSeen(),
						]
					}(),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
