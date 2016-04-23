(function(abot) {
abot.TableItemToken = {}
abot.TableItemToken.controller = function(pctrl, args) {
	var ctrl = this
	ctrl.deleteAuthToken = function(ev) {
		var c = confirm("Are you sure you want to delete this auth token?")
		if (!c) {
			ev.preventDefault()
			return
		}
		// Remove the row from the table
		this.parentNode.parentNode.remove()
		abot.request({
			url: "/api/admin/remote_tokens.json",
			method: "DELETE",
			data: { Token: args.Token },
		}).then(function(resp) {
			pctrl.props.success("Success! Deleted the auth token.")
		}, function(err) {
			pctrl.props.error("Error! Failed to delete the auth token. Err: " + err.Msg)
		})
	}
}
abot.TableItemToken.view = function(ctrl, _, args) {
	return m("tr", [
		function() {
			return m("td.x", m("a[href=#/].btn-x", {
				onclick: ctrl.deleteAuthToken,
			}, "X"))
		}(),
		m("td", "..." + args.Token.substring(args.Token.length - 6)),
		m("td.subtle", args.Email),
	])
}
})(!window.abot ? window.abot={} : window.abot);
