(function(ava) {
ava.Cards = {}
ava.Cards.controller = function() {
	// id = ??? Math.floor((Math.random() * 1000000000) + 1)
}
ava.Cards.view = function(_, props) {
	return m('div', [
		m("h3.margin-top-sm", "Credit cards"),
		m(".form-group.card", [
			m("div.table-responsive", [
				m("table.table", [
					m("thead", [
						m("tr", [
							m("th", "Type"),
							m("th", "Cardholder Name"),
							m("th", "Number"),
							m("th", "Expires"),
						])
					]),
					m("tbody", [
						props.map(function(card) {
							return m.component(ava.Card, card)
						})
					])
				]),
				m("div", [
					m("a", {
						id: '', //ctrl.props.cards.id + "-add-btn",
						class: "btn btn-sm",
						href: "/cards/new",
						config: m.route
					}, "+Add Card")
				])
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
