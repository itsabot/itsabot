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
					}, " Ava"),
				])
			]),
			m("div", {
				class: "text-right navbar-right"
			}, [
				m("a", {
					href: "/"
				}, "Home"),
				m("a", {
					href: "#"
				}, "Tour"),
				m("a", {
					href: "#"
				}, "About Us")
			])
		])
	]);
};
