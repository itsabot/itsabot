(function(abot) {
abot.ResponsePanel = {}
abot.ResponsePanel.controller = function() {
	var ctrl = this
	ctrl.route = function(msg) {
		m.route("/response_panel/conversation", {
			uid: msg.UserID,
			fid: msg.FlexID,
			fidt: msg.FlexIDType,
			off: 0,
		})
	}
	ctrl.props = {
		messages: m.prop([]),
		success: m.prop(""),
		error: m.prop(""),
	}
	abot.request({
		url: "/api/admin/conversations_need_training.json",
		method: "GET",
	}).then(function(resp) {
		ctrl.props.messages(resp)
	}, function(err) {
		ctrl.props.error(err.Msg)
	})
}
abot.ResponsePanel.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 2 }),
			m(".main", [
				m(".topbar", [
					m(".topbar-inline", "Response Panel"),
				]),
				m(".content", [
					function() {
						if (ctrl.props.error().length > 0) {
							return m(".alert.alert-danger.alert-margin", ctrl.props.error())
						}
						if (ctrl.props.success().length > 0) {
							return m(".alert.alert-success.alert-margin", ctrl.props.success())
						}
					}(),
					m("h3.top-el", "Conversations"),
					function() {
						if (ctrl.props.messages().length === 0) {
							return m("p", "No messages need responses.")
						}
					}(),
					m("table.table.table-compact", [
						ctrl.props.messages().map(function(msg) {
							return m("tr", {
								onclick: ctrl.route.bind(ctrl, msg),
							}, [
								m("td", msg.Sentence),
								m("td.subtle", abot.prettyDate(msg.CreatedAt)),
							])
						}),
					]),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
