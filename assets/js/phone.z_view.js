Phone.listView = function(list) {
	return m("table", {
		class: "table"
	}, [
		m("thead", [
			m("tr", [
				m("th", "Number")
			])
		]),
		m("tbody", [
			list.data().map(function(item) {
				var phone = new Phone();
				phone.number(item.Number);
				return m("tr", {
					"attr-id": item.Id
				}, [
					m("td", phone.format()),
					m("td", {
						class: "text-right"
					}, [
						m("img", {
							class: "icon icon-xs icon-delete",
							src: "/public/images/icon_delete.svg",
							onclick: function() {
								var c = confirm("Delete this number?");
								if (c) {
									// TODO delete from database
									console.log("not implemented");
								}
							}
						})
					])
				])
			})
		])
	]);
};
