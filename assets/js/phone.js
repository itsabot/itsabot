(function(ava) {
ava.Phone = {}
ava.Phone.view = function(_, props) {
	return m("tr", { "attr-id": props.Id }, [
		m("td", props.Number),
		m("td.text-right", [
			m("img.icon.icon-xs.icon-delete", {
				src: "/public/images/icon_delete.svg",
				onclick: function() {
					var c = confirm("Delete this number?")
					if (c) {
						// TODO delete from database
						console.log("not implemented")
					}
				}
			})
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
