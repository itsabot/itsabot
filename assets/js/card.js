(function(ava) {
ava.Card = {}
ava.Card.controller = function(props) {
	var ctrl = this
	ctrl.brandIcon = function(brand) {
		var icon
		switch(brand) {
		case "American Express", "Diners", "Discover", "JCB", "Maestro",
			"MasterCard", "PayPal", "Visa":
			var imgPath = brand.toLowerCase().replace(" ", "_")
			imgPath = "card_" + imgPath + ".svg"
			imgPath = "/public/images/" + imgPath
			icon = m("img", { src: imgPath, class: "icon-fit" })
			break
		default:
			console.log("no brand match: " + brand)
			icon = m("span", brand)
			break
		}
		return icon
	}
	ctrl.del = function() {
		var data = {
			ID: props.Id,
			UserID: parseInt(cookie.getItem("id"))
		}
		m.request({
			method: "DELETE",
			url: "/api/cards.json",
			data: data
		}).then(function() {
		}, function(err) {
			console.error(err)
		})
	}
}
ava.Card.view = function(ctrl, props) {
	return m("tr", { key: props.Id }, [
		m("td", { style: "width: 10%" }, ctrl.brandIcon(props.Brand)),
		m("td", props.CardholderName),
		m("td", {class: "subtle"}, "XXXX-" + props.Last4),
		m("td", props.ExpMonth + " / " + props.ExpYear),
		m("td", {
			class: "text-right"
		}, [
			m("img", {
				class: "icon icon-xs icon-delete",
				src: "/public/images/icon_delete.svg",
				onclick: function(ev) {
					var c = confirm("Delete this number?");
					if (c) {
						//props.del(); TODO
						ev.target.parentElement.parentElement.
							remove();
					}
				}
			})
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
