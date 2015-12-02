Card.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Card.addView(controller),
		Footer.view()
	]);
};

Card.addView = function(controller) {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-12"
			}, [
				m("h1", "Add Card")
			])
		]),
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("form", {
				class: "col-md-7 card"
			}, [
				m("div", {
					id: "card-error",
					class: "alert alert-danger hidden"
				}, controller.error()),
				m("div", {
					class: "form-horizontal"
				}, [
					m("div", {
						id: "card-number",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Card number"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "4444 0000 0000 1234",
								onchange: m.withAttr("value", controller.card.number),
								value: controller.card.number()
							})
						])
					]),
					m("div", {
						id: "card-expiry",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Expires"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "01 / 2015",
								onchange: m.withAttr("value", controller.card.expiry),
								value: controller.card.expiry()
							})
						])
					]),
					m("div", {
						id: "card-cvc",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "CVC"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "123",
								onchange: m.withAttr("value", controller.card.cvc),
								value: controller.card.cvc()
							})
						])
					]),
					m("div", {
						id: "card-name",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Cardholder name"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "Cardholder name",
								onchange: m.withAttr("value", controller.card.cardholderName),
								value: controller.card.cardholderName()
							})
						])
					]),
					m("div", {
						id: "card-name",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Billing zip"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "90210",
								onchange: m.withAttr("value", controller.card.zip5),
								value: controller.card.zip5()
							})
						])
					])
				]),
				m("div", {
					class: "text-right"
				}, [
					m("a", {
						id: "card-cancel-btn",
						class: "btn btn-sm",
						href: "/profile",
						config: m.route
					}, "Cancel"),
					m("input", {
						id: "card-save-btn",
						type: "submit",
						class: "btn btn-primary btn-sm btn-collection",
						value: controller.vm.savingText(),
						onclick: controller.saveCard,
						onsubmit: controller.saveCard
					})
				])
			])
		])
	]);
};

Card.listView = function(list) {
	return m("table", {
		class: "table"
	}, [
		m("thead", [
			m("tr", [
				m("th", "Type"),
				m("th", "Cardholder Name"),
				m("th", "Number"),
				m("th", "Expires"),
			])
		]),
		m("tbody", [
			list.data().map(function(item) {
				return m("tr", {
					"key": item.Id
				}, [
					m("td", {
						style: "width: 10%"
					}, Card.brandIcon(item.Brand)),
					m("td", item.CardholderName),
					m("td", {class: "subtle"}, "XXXX-" + item.Last4),
					m("td", item.ExpMonth + " / " + item.ExpYear),
					m("td", {
						class: "text-right"
					}, [
						m("img", {
							class: "icon icon-xs icon-delete",
							src: "/public/images/icon_delete.svg",
							onclick: function() {
								var c = confirm("Delete this number?");
								if (c) {
									// TODO delete from database and update view
									console.warn("not implemented");
								}
							}
						})
					])
				])
			})
		])
	]);
};
