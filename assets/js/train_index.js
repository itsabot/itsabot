(function(ava) {
ava.TrainIndex = {}
ava.TrainIndex.controller = function() {
	if (!ava.isLoggedIn()) {
		m.route("/login?r=" + encodeURIComponent(window.location.search))
		return
	}
	var ctrl = this
	ctrl.route = function(ev) {
		ev.preventDefault()
		var id = ev.target.parentNode.getAttribute("data-id")
		var uid = ev.target.parentNode.getAttribute("data-user-id")
		m.route("/train/" + id + "?uid=" + uid)
	}
	ctrl.props = {
		convos: m.request({
			method: "GET",
			url: "/api/messages.json"
		}),
		isTrained: m.prop(!!m.route.param("trained"))
	}
}
ava.TrainIndex.view = function(ctrl) {
	var success = null
	if (ctrl.props.isTrained()) {
		success = m(".alert.alert-success",
		       	"Success! Conversation marked as complete")
	}
	var convoList = m("h3.empty-state", "No conversations need training")
	if (!!ctrl.props.convos()) {
		convoList = m("table.table.table-bordered.table-hover", m("tbody", [
			ctrl.props.convos().map(function(conversation) {
				return m.component(ava.TrainIndexItem, conversation, ctrl)
			})
		]))
	}
	return m(".body", [
		m.component(ava.Header),
		m("#full.container", [
			m(".row.margin-top-sm", m(".col-md-12", m("h1", "Training"))),
			m(".row", m(".col-md-12.margin-top-sm", [ success, convoList ]))
		]),
		m.component(ava.Footer)
	])
}
})(!window.ava ? window.ava={} : window.ava);
