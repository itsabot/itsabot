(function(abot) {
abot.TrainIndex = {}
abot.TrainIndex.controller = function() {
	if (!abot.isLoggedIn()) {
		m.route("/login?r=" + encodeURIComponent(window.location.search))
		return
	}
	if (!abot.isTrainer()) {
		m.route("/profile")
		return
	}
	var ctrl = this
	ctrl.route = function(ev) {
		ev.preventDefault()
		var id = ev.target.parentNode.getAttribute("data-id")
		var uid = ev.target.parentNode.getAttribute("data-user-id")
		m.route("/train/" + id)
	}
	ctrl.props = {
		convos: abot.request({
			method: "GET",
			url: "/api/trainer/messages.json"
		}),
		isTrained: m.prop(!!m.route.param("trained"))
	}
}
abot.TrainIndex.view = function(ctrl) {
	var success = null
	if (ctrl.props.isTrained()) {
		success = m(".alert.alert-success",
		       	"Success! Conversation marked as complete")
	}
	var convoList = m("h3.empty-state", "No conversations need training")
	if (!!ctrl.props.convos()) {
		convoList = m("table.table", m("tbody", [
			ctrl.props.convos().map(function(conversation) {
				return m.component(abot.TrainIndexItem, conversation, ctrl)
			})
		]))
	}
	return m(".main", [
		m.component(abot.Header),
		m("h1", "Training"),
		success,
		convoList,
	])
}
})(!window.abot ? window.abot={} : window.abot);
