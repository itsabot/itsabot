var header = {};
header.view = function() {
	return m("header", {
		class: "gradient"
	}, [
		m("div", {
			class: "container"
		}, [
			m("a", {
				class: "navbar-brand",
				href: "/"
			}, [
				m("div", [
					m("img", {
						src: "/public/images/logo.svg"
					}),
					m("span", {
						class: "margin-top-xs"
					}, m.trust(" &nbsp;Ava")),
				])
			]),
			m("div", {
				class: "text-right navbar-right"
			}, [
				m("a", {
					href: "/"
				}, "Home"),
				m("a", {
					href: "/tour",
					config: m.route
				}, "Tour"),
				m("a", {
					href: "/updates",
					config: m.route
				}, "Updates")
			])
		]),
		m("div", {
			id: "content"
		})
	]);
};
