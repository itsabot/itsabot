(function(ava) {
ava.CalendarSelector = {}
ava.CalendarSelector.controller = function(props) {
	var ctrl = this
	props = props || {}
	var t = props.StartTime || new Date()
	ctrl.showForm = function(ev) {
		document.getElementById("calendar-selector-form").classList.remove("hidden")
		var tmp = Date.parse(ev.target.getAttribute("data-time"))
		var time = new Date(tmp)
		time.setMinutes(0)
		var s = time.toLocaleDateString("en-US", {
			year: "numeric",
			month: "numeric",
			day: "numeric",
			hour: "numeric",
			minute: "numeric",
			timezone: "short",
		})
		time.setMinutes(30)
		var e = time.toLocaleDateString("en-US", {
			year: "numeric",
			month: "numeric",
			day: "numeric",
			hour: "numeric",
			minute: "numeric",
			timezone: "short",
		})
		ctrl.elStartTime().value = s
		ctrl.elEndTime().value = e
		document.getElementById("calendar-selector-form-event-name").focus()
	},
	ctrl.elStartTime = function() {
		return document.getElementById("calendar-selector-form-starts")
	},
	ctrl.elEndTime = function() {
		return document.getElementById("calendar-selector-form-ends")
	},
	ctrl.newEvent = function(ev) {
		ev.preventDefault()
		ctrl.showForm(ev)
	}
	ctrl.submit = function(ev) {
		if (ev.keyCode !== 13 /* enter */) {
			return
		}
		var s = Date.parse(ctrl.elStartTime().value)
		var e = Date.parse(ctrl.elEndTime().value)
		return m.request({
			method: "POST",
			url: "/api/events.json",
			data: {
				UserID: parseInt(m.route.param("uid")),
				StartTime: s,
				EndTime: e,
			}
		})
	}
	ctrl.forward = function() {
		ctrl.props.StartTime().setHours(ctrl.props.StartTime().getHours()+24)
	}
	ctrl.backward = function() {
		ctrl.props().StartTime.setHours(ctrl.props.StartTime().getHours()-24)
	}
	ctrl.props = {
		StartTime: m.prop(t),
		BlockedTimes: m.prop(props.BlockedTimes)
	}
}
ava.CalendarSelector.view = function(ctrl) {
	var trs = []
	var hrs = []
	var colors = []
	var i = 0
	var d = new Date()
	if (ctrl.props.StartTime().getDate() === d.getDate()) {
		i = ctrl.props.StartTime().getHours()
	}
	for (var j = 0; i <= 24 && j < 24; ++i && ++j) {
		var c = ""
		if (i <= 8 || i > 17) {
			c = "gray"
		}
		var time = ""
		if (i > 12) {
			time = "" + i - 12
		} else {
			time = "" + i
		}
		if (i < 12) {
			time += "a"
		} else {
			time += "p"
		}
		if (time === "0a") {
			time = "12a"
		}
		var d = new Date()
		d.setHours(i)
		hrs.push(m("td.calendar-row." + c, time))
		colors.push(m("td.calendar-time." + c, {
			"data-id": time,
			"data-time": d,
			onclick: ctrl.newEvent
		}))
	}
	trs.push(m("tr", hrs))
	trs.push(m("tr", colors))
	return m("div", [
		// TODO convert to subcomponent (date selector)
		m("a.pull-left[href=#/]", {
			onclick: ctrl.backward.bind(ctrl),
		}, "<"),
		m("a.pull-right[href=#/]", {
			onclick: ctrl.forward.bind(ctrl),
		}, ">"),
		m(".centered", [
			ctrl.props.StartTime().toLocaleDateString("en-US", {
				weekday: "long",
				year: "numeric",
				month: "numeric",
				day: "numeric",
			})
		]),
		m("table.calendar-selector.table.table-bordered", trs),
		m(".subtle.subtle-sm.pull-right", "Timezone: Pacific"),
		m("#calendar-selector-form.margin-top-sm", { class: "hidden" }, [
			m("div", { class: "row" }, [
				m(".col-md-12", [
					m("input#calendar-selector-form-event-name", {
						class: "form-control form-white",
						placeholder: "Event name",
						onkeydown: ctrl.submit,
					})
				])
			]),
			m(".row,margin-top-xs", [
				m(".col-md-6", [
					m("input#calendar-selector-form-starts", {
						class: "form-control form-white",
						placeholder: "Starts",
						onkeydown: ctrl.submit,
					})
				]),
				m(".col-md-6", [
					m("input#calendar-selector-form-ends", {
						class: "form-control form-white",
						placeholder: "Ends",
						onkeydown: ctrl.submit,
					})
				])
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
