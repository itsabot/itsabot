var Footer = {
	controller: function() {
		return {};
	},
	view: function() {
		return m("footer", [
			m("div", {
				class: "container"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-3 border-right"
					}, [
						m("div", {
							class: "big-name"
						}, "Ava"),
						m("div", {
							class: "big-name big-name-gray"
						}, "Assistant"),
						m("div", {
							class: "margin-top-sm"
						}, m.trust("&copy; 2015 Evan Tann.")),
						m("div", "All rights reserved.")
					]),
					m("div", {
						class: "col-md-2"
					}, [
						m("div", m("a", {
							href: "/",
							config: m.route
						}, "Food")),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Travel")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Health")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Shopping")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Entertainment")
						])
					]),
					m("div", {
						class: "col-md-7 de-emphasized"
					}, [
						m("div", [
							m("a", {
								href: "/",
								config: m.route
							}, "Home")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/tour",
								config: m.route
							}, "Tour")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/updates",
								config: m.route
							}, "Updates")
						]),
					])
				])
			])
		]);
	}
};
