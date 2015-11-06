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
				href: "/",
				config: m.route
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
					href: "/",
					config: m.route
				}, "Home"),
				m("a", {
					href: "/tour",
					config: m.route
				}, "Tour"),
				m("a", {
					href: "https://medium.com/ava-updates/latest"
				}, "Updates"),
				m("a", {
					href: "/login",
					config: m.route
				}, "Log in")
			])
		]),
		m("div", {
			id: "content"
		})
	]);
};
