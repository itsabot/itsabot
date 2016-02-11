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
		ctrl.elStartTime().value = s
		document.getElementById("calendar-selector-form-event-name").focus()
	}
	ctrl.elStartTime = function() {
		return document.getElementById("calendar-selector-form-starts")
	}
	ctrl.elDuration = function() {
		return document.getElementById("calendar-selector-form-custom-duration")
	}
	ctrl.newEvent = function(ev) {
		ev.preventDefault()
		ctrl.disableSelection()
		ev.target.classList.add("blue")
		ctrl.showForm(ev)
	}
	ctrl.disableSelection = function() {
		var els = document.querySelectorAll(".calendar-time")
		for (var i = 0; i < els.length; ++i) {
			els[i].classList.remove("blue")
		}
	}
	ctrl.keySubmit = function(ev) {
		if (ev.keyCode !== 13 /* enter */) {
			return
		}
		ctrl.submit()
	}
	ctrl.submit = function() {
		var s = Date.parse(ctrl.elStartTime().value)
		var elDur = document.getElementById("calendar-selector-form-duration")
		var e = parseInt(elDur.value)
		var allDay = false
		if (elDur.selectedIndex === 7 /* All day */) {
			allDay = true
		} else if (elDur.selectedIndex === 8 /* Custom */) {
			e = parseInt(document.getElementById("calendar-selector-form-custom-duration-val").value)
		}
		var elRecur = document.getElementById("calendar-selector-form-recurring")
		var recurFreq = document.
			getElementById("calendar-selector-form-recurring-freq").
			selectedIndex
		var title = document.getElementById("calendar-selector-form-event-name").value
		return m.request({
			method: "POST",
			url: "/api/calendar/events.json",
			data: {
				Title: title,
				StartTime: s / 1000,
				DurationInMins: e,
				AllDay: allDay,
				Recurring: elRecur.checked,
				RecurringFreq: recurFreq,
				UserID: parseInt(cookie.getItem("id")),
			}
		})
	}
	ctrl.toggleCustom = function(ev) {
		if (ev.target.selectedIndex === 8 /* Custom */) {
			ctrl.elDuration().classList.remove("hidden")
		} else {
			console.log(ev.target.selectedIndex)
			ctrl.elDuration().classList.add("hidden")
		}
	}
	ctrl.forward = function() {
		ctrl.props.StartTime().setHours(ctrl.props.StartTime().getHours()+24)
	}
	ctrl.backward = function() {
		ctrl.props().StartTime.setHours(ctrl.props.StartTime().getHours()-24)
	}
	ctrl.toggleRecurring = function(ev) {
		ev.stopImmediatePropagation()
		var sel = document.getElementById("calendar-selector-form-recurring-freq")
		if (sel.hasAttribute("readOnly")) {
			sel.removeAttribute("readOnly")
		} else {
			sel.setAttribute("readOnly", true)
		}
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
	return m(".card", [
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
						onkeydown: ctrl.keySubmit,
					})
				])
			]),
			m(".row margin-top-xs", [
				m(".col-md-6", [
					m("input#calendar-selector-form-starts", {
						class: "form-control form-white",
						placeholder: "Starts",
						onkeydown: ctrl.keySubmit,
						onchange: ctrl.disableSelection,
					})
				]),
				m(".col-md-6", [
					m("select#calendar-selector-form-duration", {
						class: "form-control form-white",
						onkeydown: ctrl.keySubmit,
						onchange: ctrl.toggleCustom
					}, [
						m("option", { value: 15 }, "15 minutes"),
						m("option", { value: 30 }, "30 minutes"),
						m("option", { value: 45 }, "45 minutes"),
						m("option", { value: 60 }, "1 hour"),
						m("option", { value: 120 }, "2 hours"),
						m("option", { value: 240 }, "4 hours"),
						m("option", { value: 480 }, "8 hours"),
						m("option", { value: 0 }, "All day"),
						m("option", { value: 0 }, "Custom"),
					])
				])
			]),
			m(".row.margin-top-xs.hidden", {
				id: "calendar-selector-form-custom-duration"
			}, [
				m(".col-md-12", [
					m("input#calendar-selector-form-custom-duration-val", {
						class: "form-control form-white",
						placeholder: "Length of event in minutes",
						type: "number",
					})
				])
			]),
			m(".row", [
				m(".col-md-6", [
					m(".checkbox", [
						m("label", [
							m("input#calendar-selector-form-recurring", {
								type: "checkbox",
								value: true,
								onclick: ctrl.toggleRecurring,
							}),
							m("span", "Recurring")
						])
					])
				]),
				m(".col-md-6.margin-top-xs", [
					m("select#calendar-selector-form-recurring-freq", {
						class: "form-control form-white",
						readOnly: true,
					}, [
						m("option", { value: "once" }, "Once"),
						m("option", { value: "daily" }, "Daily"),
						m("option", { value: "weekly" }, "Weekly"),
						m("option", { value: "monthly" }, "Monthly"),
						m("option", { value: "yearly" }, "Yearly"),
					])
				])
			]),
			m(".row", [
				m(".col-md-12.margin-top-xs", [
					m("button.btn.btn-primary.btn-sm", {
						onclick: ctrl.submit,
					}, "Create")
				])
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
