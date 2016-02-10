(function(ava) {
ava.Index = {}
ava.Index.vm = {
	showEarlyAccess: function() {
		document.getElementById("btns").classList.add("hidden")
		document.getElementById("earlyAccess").classList.remove("hidden")
		setTimeout(function() {
			document.getElementById("earlyAccess").classList.add("fade-in")
		}, 300)
	}
}
ava.Index.view = function() {
	return m("div", [
		m("div", { class: "gradient gradient-big gradient-bright" }, [
			m("div", { class: "jumbo-container" }, [
				m.component(ava.Header),
				m("div", { class: "jumbo container" }, [
					m("div", { class: "col-md-8" }, [
						m("h1", "Meet Ava."),
						m("br"),
						m("h1", "Your new assistant."),
						m("p", {
							id: "earlyAccess",
							class: "hidden fade"
						}, [
							'Get early access:',
							m("h3", 'Text Ava at ', [
								m("strong", {
									class: "phone"
								}, "(424) 297-1568"),
								' and say "Hi!"'
							])
						]),
						m("div", {
							id: "btns-container"
						}, [
							m("p", "Get early access to the world's most advanced digital assistant."),
							m("div", {
								id: "btns"
							}, [
								m("a", {
									class: "btn",
									href: "/tour",
									config: m.route
								}, "Take a tour"),
								m("a", {
									class: "btn btn-green",
									onclick: ava.Index.vm.showEarlyAccess
								}, "Get early access")
							])
						])
					]),
					m("div", {
						class: "col-md-4"
					}, [
						m("img", {
							class: "img-big",
							src: "/public/images/iphone.png"
						})
					])
				])
			])
		]),
		m("div", { class: "container body-container" }, [
			m("div", { class: "row" }, [
				m("div", { class: "col-md-1 margin-bottom" }, [
					m("div", {
						class: "label label-primary"
					}, "New")
				]),
				m("div", { class: "col-md-4" }, [
					m("p", "Car trouble? Ava now finds recommended mechanic and tow services nearby.")
				]),
				m("div", { class: "col-md-2" }, [
					m("a", {
						class: "bold",
						href: "https://medium.com/@egtann/car-mechanic-1af70923eb19#.o7htx32u7"
					}, m.trust("Read more &nbsp; &#9654;"))
				])
			])
		]),
		m.component(ava.Footer)
	])
}
})(!window.ava ? window.ava={} : window.ava);
