(function(ava) {
ava.Searchbox = {}
ava.Searchbox.controller = function(props, pctrl) {
	var ctrl = this
	ctrl.update = function(ev) {
		var query = ev.target.value
		ctrl.props.Username(query)
		if (query.length < 3) {
			// The server uses trigram indexes in Postgres, so input must be at
			// least 3 characters
			ctrl.props.SearchResults([])
			return
		}
		m.request({
			method: "GET",
			url: "/api/contacts/search.json",
			data: {
				Query: query,
				UserID: parseInt(cookie.getItem("id")),
			},
		}).then(function(resp) {
			resp = resp || []
			if (resp.length > 0) {
				ev.target.nextSibling.classList.remove("hidden")
			} else {
				ev.target.nextSibling.classList.add("hidden")
			}
			ctrl.props.SearchResults(resp)
		}, function(err) {
			console.error(err)
		})
	}
	ctrl.selectResult = function(result) {
		ctrl.props.SearchResults([])
		ctrl.props.Username(result.Name + " - " + result.Contact)
		if (pctrl.props.Contact != null) {
			pctrl.props.Contact(result.Contact)
			pctrl.props.ContactMethod(result.ContactMethod)
		}
	}
	ctrl.props = {
		SearchResults: m.prop([]),
		Username: m.prop(props.Username()),
	}
}
ava.Searchbox.view = function(ctrl, props, pctrl) {
	console.log(ctrl.props.Username())
	return m("div", [
		m("input[type=text][placeholder=To:].form-control.form-white", {
			disabled: props.UsernameDisabled,
			value: ctrl.props.Username(),
			oninput: ctrl.update,
		}),
		m("ol.list-unstyled.autocomplete.hidden",
			ctrl.props.SearchResults().map(function(item, i) {
				if (item.Phone !== null) {
					item.Contact = item.Phone
					item.ContactMethod = "Phone"
				} else if (item.Email !== null) {
					item.Contact = item.Email
					item.ContactMethod = "Email"
				} else {
					// No way to contact user. Should never happen
					return
				}
				return m("li", {
					key: i,
					onclick: ctrl.selectResult.bind(ctrl, item),
				}, [
					m("strong", item.Name),
					m("span", " - " + item.Contact),
				])
			})
		)
	])
}
})(!window.ava ? window.ava={} : window.ava);
