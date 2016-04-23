(function(abot) {
abot.TableItemUser = {}
abot.TableItemUser.controller = function(pctrl, args) {
	var ctrl = this
	ctrl.removeAdmin = function(ev) {
		var c = confirm("Are you sure you want to remove admin permissions from " +
						args.Email + "?")
		if (!c) {
			ev.preventDefault()	
			return
		}
		// Remove the row from the table
		this.parentNode.parentNode.remove()
		abot.request({
			url: "/api/admins.json",
			method: "PUT",
			data: {
				ID: parseInt(args.ID),
				Admin: false,
			},
		}).then(function() {
			pctrl.props.success("Success! Removed admin permissions from " + args.Email)
		}, function(err) {
			pctrl.props.error("Error! Failed to remove admin. Err: " + err.Msg)
		})
	}
}
abot.TableItemUser.view = function(ctrl, _, args) {
	var trainer
	if (args.Trainer) {
		// Leading space is important
		trainer = " badge-highlighted"
	}
	var manager
	if (args.ManageTeam) {
		manager = " badge-highlighted"
	}
	return m("tr", [
		function() {
			if (args.Email === Cookies.get("email")) {
				return m("td")
			}
			return m("td.x", m("a[href=#/].btn-x", {
				onclick: ctrl.removeAdmin,
			}, "X"))
		}(),
		m("td", args.Name),
		m("td.subtle", args.Email),
	])
}
})(!window.abot ? window.abot={} : window.abot);
