"use strict";

window.onload = function() {
	document.onclick = function(event) {
		if (event.target.classList.contains("pathText")) {
			let p = event.target;
			let ptxt = p.textContent;
			let main = document.querySelector(".main");
			if (ptxt.startsWith(main.dataset.copyPathRemapFrom)) {
				ptxt = ptxt.replace(main.dataset.copyPathRemapFrom, main.dataset.copyPathRemapTo);
			}
			let succeeded = function() {
				printStatus("path copied: " + ptxt);
			}
			let failed = function() {
				printStatus("failed to copy path");
			}
			navigator.clipboard.writeText(ptxt).then(succeeded, failed);
			return;
		}
		let counter = event.target.closest(".statusCounter");
		if (counter) {
			let group = counter.closest(".statusGroup");
			let sum = group.closest(".statusSummary");
			let entType = group.dataset.entryType;
			let status = counter.dataset.status;
			if (sum.dataset.selected == "1" && sum.dataset.selectedEntryType == entType && sum.dataset.selectedStatus == status) {
				sum.dataset.selected = "";
			} else {
				sum.dataset.selected = "1";
				sum.dataset.selectedEntryType = entType;
				sum.dataset.selectedStatus = status;
			}
			let forTypes = document.querySelectorAll(".subEntryListForType");
			for (let forType of forTypes) {
				let typ = forType.dataset.entryType;
				for (let ent of forType.querySelectorAll(".subEntry")) {
					if (sum.dataset.selected != "1") {
						ent.style.removeProperty("display"); // show every entry
						continue;
					}
					if (forType.dataset.entryType != entType) {
						ent.style.display = "none"
						ent.classList.remove("selected");
						continue;
					}
					let dot = ent.querySelector(".statusDot");
					if (dot != null) {
						if (dot.dataset.value != sum.dataset.selectedStatus) {
							ent.style.display = "none";
							ent.classList.remove("selected");
							continue;
						}
					} else {
						// Should work for entries of type that doesn't have status.
						if (sum.dataset.selectedStatus != "") {
							ent.style.display = "none";
							ent.classList.remove("selected");
							continue;
						}
					}
					ent.style.removeProperty("display"); // show
				}
				let nTotal = 0;
				for (let cnt of forType.querySelectorAll(".subEntryListContainer")) {
					let n = 0;
					for (let ent of cnt.querySelectorAll(".subEntry")) {
						if (ent.style.display != "none") {
							n++;
							nTotal++;
						}
					}
					let count = cnt.querySelector(".subEntryListFromCount");
					if (count) {
						count.innerText = "(" + String(n) + ")";
					}
					if (n == 0) {
						cnt.style.display = "none";
					} else {
						cnt.style.removeProperty("display");
					}
				}
				let bar = forType.querySelector(".subEntryTypeBar");
				let typeCount = bar.querySelector(".subEntryTypeCount");
				if (typeCount) {
					typeCount.innerText = "(" + String(nTotal) + ")";
				}
				if (nTotal == 0) {
					bar.style.display = "none";
				} else {
					bar.style.removeProperty("display");
				}
			}
			return;
		}
		let options = event.target.closest(".subEntryListOptions");
		if (options) {
			let opt = event.target.closest(".subEntryListOption.expandOption");
			if (opt) {
				if (opt.dataset.disabled == "1") {
					// don't shrink.
					return;
				}
				if (opt.dataset.value == "") {
					opt.dataset.value = "1";
				} else {
					opt.dataset.value = "";
				}
				let area = event.target.closest(".subEntryArea");
				let conts = area.querySelectorAll(".subEntryListContainer");
				for (let c of conts) {
					c.dataset.expanded = opt.dataset.value;
				}
				let req = new XMLHttpRequest();
				let formData = new FormData();
				formData.append("update_search_result_expand", "1");
				formData.append("expand", opt.dataset.value);
				req.open("post", "/api/update-user-setting");
				req.onerror = function() {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				req.onload = function() {
					if (req.status != 200) {
						printErrorStatus(req.responseText);
						return;
					}
				}
				req.send(formData);
				return;
			}
			opt = event.target.closest(".subEntryListOption.viewOption");
			if (opt) {
				if (opt.dataset.value != "thumbnail") {
					opt.dataset.value = "thumbnail";
				} else {
					opt.dataset.value = "";
				}
				for (let ent of document.querySelectorAll(".subEntry")) {
					if (opt.dataset.value == "thumbnail") {
						ent.classList.remove("expanded");
					} else {
						ent.classList.add("expanded");
					}
				}
				let area = event.target.closest(".subEntryArea");
				area.dataset.view = opt.dataset.value;
				let req = new XMLHttpRequest();
				let formData = new FormData();
				formData.append("update_search_view", "1");
				formData.append("view", opt.dataset.value);
				req.open("post", "/api/update-user-setting");
				req.onerror = function() {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				req.onload = function() {
					if (req.status != 200) {
						printErrorStatus(req.responseText);
						return;
					}
				}
				req.send(formData);
				return;
			}
		}
		let expander = event.target.closest(".propertyExpander");
		if (expander) {
			let cont = document.querySelector(".mainEntryInfoContainer");
			if (cont.dataset.showHidden == "") {
				cont.dataset.showHidden = "1";
			} else {
				cont.dataset.showHidden = "";
			}
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("update_entry_page_show_hidden_property", "1");
			formData.append("show_hidden", cont.dataset.showHidden);
			req.open("post", "/api/update-user-setting");
			req.onerror = function() {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status != 200) {
					printErrorStatus(req.responseText);
					return;
				}
			}
			req.send(formData);
			return;
		}
		expander = event.target.closest(".thumbnailViewExpander");
		if (expander) {
			let thisEnt = expander.closest(".subEntry");
			let expand = !thisEnt.classList.contains("expanded");
			let selected = {}
			for (let ent of document.querySelectorAll(".subEntry")) {
				if (ent.classList.contains("selected")) {
					selected[ent.dataset.entryPath] = true;
				}
			}
			if (Object.keys(selected).length != 0) {
				if (!selected[thisEnt.dataset.entryPath]) {
					printErrorStatus("entry not in selection: " + thisEnt.dataset.entryPath);
					return;
				}
			} else {
				selected[thisEnt.dataset.entryPath] = true;
			}
			for (let ent of document.querySelectorAll(".subEntry")) {
				if (selected[ent.dataset.entryPath] != null) {
					if (expand) {
						ent.classList.add("expanded");
					} else {
						ent.classList.remove("expanded");
					}
				}
			}
			return;
		}
		expander = event.target.classList.contains("subEntryListExpander");
		if (expander) {
			let cont = event.target.closest(".subEntryListContainer");
			let conts = [cont]
			if (event.shiftKey) {
				let area = event.target.closest(".subEntryArea");
				conts = area.querySelectorAll(".subEntryListContainer");
			}
			let expanded = cont.dataset.expanded;
			for (let c of conts) {
				if (!expanded) {
					c.dataset.expanded = "1"
				} else {
					c.dataset.expanded = ""
				}
			}
		}
		let hide = false;
		let handle = event.target.closest(".statusSelector, .updatePropertyPopup");
		if (handle != null) {
			hide = true;
			let mainDiv = document.querySelector(".main");
			let fn = function() {
				if (handle.classList.contains("statusSelector")) {
					let sel = handle;
					let thisEnt = sel.closest(".entry");
					let entPath = thisEnt.dataset.entryPath;
					let entType = sel.dataset.entryType;
					let popup = document.querySelector(`.updatePropertyPopup[data-entry-type="${entType}"]`);
					if (popup == null) {
						printErrorStatus("'possible_status' global not defined for '" + entType + "' entry type");
						return;
					}
					if (thisEnt.classList.contains("subEntry")) {
						let editMode = subEntArea.classList.contains("editMode");
						if (!editMode) {
							return;
						}
						let selectedEnts = document.querySelectorAll(".subEntry.selected");
						if (selectedEnts.length != 0) {
							let inSel = false;
							for (let ent of selectedEnts) {
								if (entPath == ent.dataset.entryPath) {
									inSel = true;
									break;
								}
							}
							if (!inSel) {
								sel.dataset.popupAttached = "";
								mainDiv.dataset.currentSelectStatusMenu = "";
								printErrorStatus("entry not in selection: " + entPath);
								return;
							}
						}
					}
					popup.dataset.entryPath = entPath;
					popup.dataset.sub = sel.dataset.sub;
					if (sel.dataset.popupAttached == "1") {
						sel.dataset.popupAttached = "";
						mainDiv.dataset.currentSelectStatusMenu = "";
						hide = true;
						return;
					}
					mainDiv.dataset.currentSelectStatusMenu = sel.dataset.entryType;
					let attached = document.querySelector(`.statusSelector[data-popup-attached="1"]`)
					if (attached) {
						attached.dataset.popupAttached = "";
					}
					sel.dataset.popupAttached = "1";
					let nameInput = popup.querySelector(".propertyPickerName");
					reloadPropertyPicker(popup, nameInput.value.trim());
					// slight adjust of the popup position to make statusDots aligned.
					let right = sel.closest(".right");
					let offset = offsetFrom(sel, right);
					popup.style.left = String(offset.left - 6) + "px";
					popup.style.top = String(offset.top + sel.offsetHeight + 4) + "px";
				} else {
					let popup = handle;
					let thisEnt = document.querySelector(`.entry[data-entry-path="${popup.dataset.entryPath}"]`);
					let entPath = thisEnt.dataset.entryPath;
					let item = event.target.closest(".selectStatusMenuItem");
					if (item != null) {
						let selectedEnts = document.querySelectorAll(".subEntry.selected");
						if (selectedEnts.length != 0) {
							let inSel = false;
							for (let ent of selectedEnts) {
								if (entPath == ent.dataset.entryPath) {
									inSel = true;
									break;
								}
							}
							if (!inSel) {
								mainDiv.dataset.currentSelectStatusMenu = "";
								printErrorStatus("entry not in selection: " + entPath);
								return;
							}
						}
						if (selectedEnts.length == 0) {
							selectedEnts = [thisEnt];
						}
						let sub = popup.dataset.sub;
						let req = new XMLHttpRequest();
						let formData = new FormData();
						for (let ent of selectedEnts) {
							let dot = ent.querySelector(`.statusSelector[data-sub="${sub}"]`);
							if (!dot) {
								continue;
							}
							let path = ent.dataset.entryPath;
							if (sub != "") {
								path += "/" + sub;
							}
							formData.append("path", path);
						}
						formData.append("name", "status");
						formData.append("value", item.dataset.value);
						req.open("post", "/api/update-property");
						req.send(formData);
						req.onload = function() {
							if (req.status == 200) {
								for (let ent of selectedEnts) {
									let dot = ent.querySelector(`.statusSelector[data-sub="${sub}"]`);
									if (!dot) {
										continue;
									}
									dot.dataset.value = item.dataset.value;
								}
								mainDiv.dataset.currentSelectStatusMenu = "";
							} else {
								printErrorStatus(req.responseText);
							}
						}
						req.onerror = function(err) {
							printErrorStatus("network error occurred. please check whether the server is down.");
						}
					}
				}
			}
			fn()
		} else {
			let mainDiv = document.querySelector(".main");
			if (mainDiv.dataset.currentSelectStatusMenu != "") {
				mainDiv.dataset.currentSelectStatusMenu = "";
				let attached = document.querySelector(`.statusSelector[data-popup-attached="1"]`)
				if (attached) {
					attached.dataset.popupAttached = "";
				}
				hide = true;
			}
		}
		if (event.target.closest(".assigneeInput") == null) {
			let userMenu = document.getElementById("userAutoCompleteMenu");
			if (!userMenu.classList.contains("invisible")) {
				userMenu.replaceChildren();
				userMenu.classList.add("invisible");
				hide = true;
			}
		}
		if (event.target.closest(".infoContextMenuLoader") == null) {
			let infoMenu = document.getElementById("infoContextMenu");
			if (currentContextMenuLoader != null) {
				infoMenu.classList.add("invisible");
				currentContextMenuLoader = null;
				hide = true;
			}
		}
		if (hide) {
			return;
		}
		if (!event.target.closest(".infoAdder, .infoTitle, #footer") || event.target.closest("#footer .closeButton")) {
			let active = document.querySelector(".infoTitle.active");
			if (active) {
				active.classList.remove("active");
			}
			hide = hideInfoModifier();
		}
		if (event.target.closest(".grandSubAdderLoader")) {
			let addingArea = document.querySelector(".grandSubArea.adding");
			if (addingArea != null) {
				addingArea.classList.remove("adding");
			}
			let subEnt = event.target.closest(".subEntry");
			if (document.querySelectorAll(".subEntry.selected").length != 0 && !subEnt.classList.contains("selected")) {
				printErrorStatus("entry not in selection: " + subEnt.dataset.entryPath);
				return;
			}
			let area = event.target.closest(".grandSubArea");
			area.classList.add("adding");
			let input = area.querySelector(".grandSubAdderInput");
			// move cursor to end of input content
			let sel = window.getSelection();
		    sel.selectAllChildren(input);
		    sel.collapseToEnd();
		} else if (event.target.closest(".grandSubAdder") == null) {
			let addingArea = document.querySelector(".grandSubArea.adding");
			if (addingArea != null) {
				addingArea.classList.remove("adding");
				hide = true;
			}
		}
		if (hide) {
			return;
		}
		if (event.target.closest(".subEntryList, #footer") == null) {
			let subEntArea = document.querySelector(".subEntryArea");
			if (subEntArea.classList.contains("editMode")) {
				let selEnts = document.querySelectorAll(".subEntry.selected");
				if (selEnts.length == 0) {
					subEntArea.classList.remove("editMode");
					removeClass(subEntArea, "lastClicked");
					removeClass(subEntArea, "temporary");
					printStatus("normal mode");
					return;
				}
				for (let ent of selEnts) {
					ent.classList.remove("selected");
				}
				removeClass(subEntArea, "lastClicked");
				removeClass(subEntArea, "temporary");
				printStatus("no entry selected");
			}
		}
	}
	document.onkeydown = function(event) {
		let ctrlPressed = event.ctrlKey || event.metaKey;
		if (event.code == "Escape") {
			// Will close floating UIs first, if any exists.
			let hide = false;
			let mainDiv = document.querySelector(".main");
			if (mainDiv.dataset.currentSelectStatusMenu != "") {
				mainDiv.dataset.currentSelectStatusMenu = "";
				hide = true;
			}
			let userMenu = document.getElementById("userAutoCompleteMenu");
			if (!userMenu.classList.contains("invisible")) {
				userMenu.replaceChildren();
				userMenu.classList.add("invisible");
				hide = true;
			}
			let infoMenu = document.getElementById("infoContextMenu");
			if (currentContextMenuLoader != null) {
				infoMenu.classList.add("invisible");
				currentContextMenuLoader = null;
				hide = true;
			}
			if (hide) {
				return;
			}
			hide = hideInfoModifier();
			if (hide) {
				return;
			}
			let subEntArea = document.querySelector(".subEntryArea");
			if (subEntArea.classList.contains("editMode")) {
				let selEnts = document.querySelectorAll(".subEntry.selected");
				if (selEnts.length == 0) {
					subEntArea.classList.remove("editMode");
					removeClass(subEntArea, "lastClicked");
					removeClass(subEntArea, "temporary");
					printStatus("normal mode");
					return;
				}
				for (let ent of selEnts) {
					ent.classList.remove("selected");
				}
				removeClass(subEntArea, "lastClicked");
				removeClass(subEntArea, "temporary");
				printStatus("no entry selected");
				return;
			}
			return;
		}
		if (event.target.closest(".propertyPickerValue")) {
			if ((ctrlPressed && event.code == "Enter") || event.code == "NumpadEnter") {
				let popup = event.target.closest(".updatePropertyPopup");
				let nameInput = popup.querySelector(".propertyPickerName");
				let valueInput = popup.querySelector(".propertyPickerValue");
				let prop = nameInput.value.trim();
				if (prop == "") {
					return;
				}
				let mainDiv = document.querySelector(".main");
				let entPath = popup.dataset.entryPath;
				let sub = popup.dataset.sub;
				let selEnts = document.querySelectorAll(".subEntry.selected");
				let paths = [];
				if (selEnts.length == 0) {
					paths.push(entPath);
				} else {
					for (let ent of selEnts) {
						let path = ent.dataset.entryPath;
						if (sub != "") {
							if (ent.querySelector(`.grandSubEntry[data-sub="${sub}"]`) == null) {
								continue
							}
							path += "/" + sub;
						}
						paths.push(path);
					}
				}
				let req = new XMLHttpRequest();
				let formData = new FormData();
				for (let path of paths) {
					formData.append("path", path);
				}
				formData.append("name", prop);
				formData.append("value", valueInput.value.trim());
				req.open("post", "/api/update-property");
				req.send(formData);
				req.onerror = function() {
					nameInput.dataset.error = "1";
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				req.onload = function() {
					if (req.status != 200) {
						nameInput.dataset.error = "1";
						printErrorStatus(req.responseText);
						return;
					}
					nameInput.dataset.error = "";
					nameInput.dataset.modified = "";
					printStatus("done");
				}
			}
			return;
		}
		if (event.code == "KeyA") {
			if (!ctrlPressed) {
				return;
			}
			let userEditables = ["textarea", "input"];
			if (userEditables.includes(event.target.tagName.toLowerCase())) {
				return;
			}
			let subEntArea = document.querySelector(".subEntryArea");
			if (!subEntArea.classList.contains("editMode")) {
				return;
			}
			event.preventDefault();
			let nVis = 0;
			let selEnts = document.querySelectorAll(".subEntry");
			for (let ent of selEnts) {
				// Wierd way of checking it's visibility, but it is what it is.
				let vis = ent.offsetWidth > 0 || ent.offsetHeight > 0;
				if (vis) {
					nVis++
					ent.classList.add("selected");
				}
			}
			removeClass(subEntArea, "lastClicked");
			removeClass(subEntArea, "temporary");
			let what = "";
			let entry = "entry"
			if (nVis == 0) {
				what = "no entry";
			} else if (nVis == 1) {
				what = "1 entry";
			} else {
				what = String(nVis) + " entries";
			}
			printStatus(what + " selected");
			return;
		}
	}
	document.onchange = function(event) {
		if (event.target.closest(".propertyPickerName")) {
			let popup = event.target.closest(".updatePropertyPopup");
			let nameInput = popup.querySelector(".propertyPickerName");
			nameInput.dataset.value = nameInput.value;
			let valueInput = popup.querySelector(".propertyPickerValue");
			let prop = nameInput.value.trim();
			let req = new XMLHttpRequest();
			let formData = new FormData();
			let entType = popup.dataset.entryType;
			if (entType == "") {
				printErrorStatus("entry type should not be empty.");
				return;
			}
			formData.append("update_picked_property", "1");
			formData.append("entry_type", entType);
			formData.append("picked_property", prop);
			req.open("post", "/api/update-user-setting");
			req.send(formData);
			req.onerror = function() {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status != 200) {
					printErrorStatus(req.responseText);
					return;
				}
				reloadPropertyPicker(popup, prop);
			}
			return;
		}
		let opt = event.target.closest(".subEntryListOption.groupByOption");
		if (opt) {
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("update_entry_group_by", "1");
			formData.append("group_by", opt.value);
			req.open("post", "/api/update-user-setting");
			req.onerror = function() {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status != 200) {
					printErrorStatus(req.responseText);
					return;
				}
				location.reload();
			}
			req.send(formData);
			return;
		}
	}
	document.oninput = function() {
		if (event.target.closest(".propertyPickerValue")) {
			let popup = event.target.closest(".updatePropertyPopup");
			let nameInput = popup.querySelector(".propertyPickerName");
			nameInput.dataset.error = "";
			nameInput.dataset.modified = "1";
		}
	}
	let allInputs = document.getElementsByTagName("input");
	for (let input of allInputs) {
		input.autocomplete = "off";
	}
	let inputs = document.getElementsByClassName("valueInput");
	for (let input of inputs) {
		input.onkeydown = function(ev) {
			if (((ev.ctrlKey || ev.metaKey) && ev.code == "Enter") || ev.code == "NumpadEnter") {
				submitUpdaterOrAdder(ev, input);
			}
		}
		input.parentElement.onsubmit = function(ev) {
			submitUpdaterOrAdder(ev, input);
		}
	}
	for (let input of inputs) {
		input.oninput = function() {
			resizeTextArea(input);
		}
	}
	let uploadExcelInput = document.getElementById("uploadExcelInput");
	if (uploadExcelInput != null) {
		uploadExcelInput.onchange = function() {
			let uploadExcelForm = document.getElementById("uploadExcelForm");
			uploadExcelForm.submit();
		}
	}
	let pinnedPaths = document.getElementsByClassName("pinnedPathLink");
	for (let pp of pinnedPaths) {
		pp.onclick = function(event) {
			window.location.href = pp.dataset.link;
		}
		pp.ondragstart = function(event) {
			event.dataTransfer.effectAllowed = "move";
			let zone = document.getElementById("pinnedPathDropZone");
			// TODO: extract common code to drag-drop for zones as functions.
			zone.ondragover = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "move";
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let p of paths) {
					if (p == pp) {
						continue;
					}
					let rect = p.getBoundingClientRect();
					if (event.clientY < rect.top + rect.height/2) {
						p.parentNode.insertBefore(pp, p);
						break;
					}
					if (p.nextElementSibling == null) {
						p.parentNode.appendChild(pp, p);
						break;
					}
				}
			}
			zone.ondragleave = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "none";
			}
			zone.ondrop = function(event) {
				let at = -1;
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let i = 0; i < paths.length; i++) {
					let p = paths[i];
					if (p == pp) {
						at = i;
						break;
					}
				}
				if (at == -1) {
					console.log("unable to find drop target from pinned paths");
					return;
				}
				updatePinnedPath(pp.innerText, at);
			}
			let del = document.getElementById("pinnedPathDeleteButton");
			del.classList.remove("nodisplay");
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.style.color = "#A22";
				del.style.border = "1px solid #A22";
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.style.color = "#DDD";
				del.style.border = "1px solid #DDD";
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				updatePinnedPath(pp.innerText, -1);
			}
		}
		pp.ondragend = function(event) {
			let zone = document.getElementById("pinnedPathDropZone");
			removeDragDropEvents(zone);
			let del = document.getElementById("pinnedPathDeleteButton");
			removeDragDropEvents(del);
			del.classList.add("nodisplay");
			if (event.dataTransfer.dropEffect == "none") {
				let curIdx = -1;
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let i = 0; i < paths.length; i++) {
					let p = paths[i];
					if (p == pp) {
						curIdx = i;
						break;
					}
				}
				if (curIdx == -1) {
					console.log("unable to find drop target from pinnedpaths");
					return;
				}
				let origIdx = parseInt(pp.dataset.idx);
				if (curIdx == origIdx) {
					return;
				}
				if (curIdx > origIdx) {
					pp.parentNode.insertBefore(pp, pp.parentNode.children[origIdx]);
					return;
				}
				if (curIdx < origIdx) {
					if (origIdx+1 == paths.length) {
						pp.parentNode.appendChild(pp);
						return;
					}
					pp.parentNode.insertBefore(pp, pp.parentNode.children[origIdx+1]);
					return;
				}
			}
		}
	}
	let quickSearches = document.getElementsByClassName("quickSearchLink");
	for (let qs of quickSearches) {
		qs.onclick = function(event) {
			// I've had hard time when I drag quickSearchLink while it is 'a' tag.
			// At first glance qs.ondragstart seemed to work consitently, then the link is clicked instead.
			// Hope I got peace by making quickSearchLink 'div'.
			window.location.href = qs.dataset.link;
			return;
		}
		qs.ondragstart = function(event) {
			event.dataTransfer.effectAllowed = "move";
			let zone = document.getElementById("quickSearchDropZone");
			zone.ondragover = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "move";
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let s of searches) {
					if (s == qs) {
						continue;
					}
					let rect = s.getBoundingClientRect();
					if (event.clientY < rect.top || rect.top + rect.height <= event.clientY) {
						continue;
					}
					if (event.clientX < rect.left + rect.width) {
						if (event.clientX < rect.left + rect.width/2) {
							s.parentNode.insertBefore(qs, s);
							break;
						}
						let next = s.nextElementSibling;
						if (next == null) {
							s.parentNode.appendChild(qs, s);
							break;
						}
						s.parentNode.insertBefore(qs, next);
						break;
					}
				}
			}
			zone.ondragleave = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "none";
			}
			zone.ondrop = function(event) {
				let at = -1;
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let i = 0; i < searches.length; i++) {
					let s = searches[i];
					if (s == qs) {
						at = i;
						break;
					}
				}
				if (at == -1) {
					console.log("unable to find drop target from quicksearches");
					return;
				}
				updateQuickSearch(qs.innerText, at);
			}
			let del = document.getElementById("quickSearchDeleteButton");
			del.classList.remove("nodisplay");
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.style.color = "#A22";
				del.style.border = "1px solid #A22";
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.style.color = "#DDD";
				del.style.border = "1px solid #DDD";
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				updateQuickSearch(qs.innerText, -1);
			}
		}
		qs.ondragend = function(event) {
			let zone = document.getElementById("quickSearchDropZone");
			removeDragDropEvents(zone);
			let del = document.getElementById("quickSearchDeleteButton");
			removeDragDropEvents(del);
			del.classList.add("nodisplay");
			if (event.dataTransfer.dropEffect == "none") {
				let curIdx = -1;
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let i = 0; i < searches.length; i++) {
					let s = searches[i];
					if (s == qs) {
						curIdx = i;
						break;
					}
				}
				if (curIdx == -1) {
					console.log("unable to find drop target from quicksearches");
					return;
				}
				let origIdx = parseInt(qs.dataset.idx);
				if (curIdx == origIdx) {
					return;
				}
				if (curIdx > origIdx) {
					qs.parentNode.insertBefore(qs, qs.parentNode.children[origIdx]);
					return;
				}
				if (curIdx < origIdx) {
					if (origIdx+1 == searches.length) {
						qs.parentNode.appendChild(qs);
						return;
					}
					qs.parentNode.insertBefore(qs, qs.parentNode.children[origIdx+1]);
					return;
				}
			}
		}
	}
	let subEntArea = document.querySelector(".subEntryArea");
	let alreadyHandled = false;
	let mousedownId = 0;
	let subEntries = document.getElementsByClassName("subEntry");
	for (let ent of subEntries) {
		ent.onmousedown = function(event) {
			if (event.button != 0) {
				// not a left mouse button
				return;
			}
			alreadyHandled = false;
			if (subEntArea.classList.contains("editMode")) {
				// prevent text selection
				// TODO: mouse dragging should be prevented as well
				if (event.shiftKey) {
					event.preventDefault();
					return;
				}
				return;
			}
			// Treat alt+click as a command to turn on editMode immediately.
			// NOTE: ctrlKey and metaKey are also binded to see which is better key binding.
			// I might eventually remove altKey binding.
			if (event.altKey || event.ctrlKey || event.metaKey) {
				subEntArea.classList.add("editMode");
				printStatus("edit mode");
				alreadyHandled = true;
				return;
			}
			// Two conditions should met to turn on editMode.
			// User holding mouse down for reasonable duration,
			// and mouse movement should be relatively small. (to distinguish it from text selection)
			function matchMousedownId(n) {
				// Do not merge this function into setTimeout function,
				// It will not working correctly because of mousedownId variable scope.
				return mousedownId == n
			}
			function distance(dx, dy) {
				return Math.sqrt(Math.pow(dx, 2) + Math.pow(dy, 2))
			}
			// ids aren't usually typed as float, but it's ok here.
			mousedownId = Math.random();
			let id = mousedownId;
			let x1 = event.clientX;
			let y1 = event.clientY;
			let x2 = x1;
			let y2 = y1;
			ent.onmousemove = function(ev) {
				x2 = ev.clientX;
				y2 = ev.clientY;
			}
			setTimeout(function() {
				ent.onmousemove = function() {}
				if (!matchMousedownId(id)) {
					return;
				};
				alreadyHandled = true;
				if (distance(x2-x1, y2-y1) > 5) {
					return;
				}
				subEntArea.classList.add("editMode");
				printStatus("edit mode");
			}, 500)
		}
		ent.onmouseup = function(event) {
			// Mouse would be up-ed inside of ".statusDot, .infoTitle, .assigneeInput". It is ok.
			mousedownId = 0;
		}
		ent.onclick = function(event) {
			if (event.target.closest(".statusDot, .summaryDot, .infoTitle, .assigneeInput, .pathText, .thumbnailViewExpander") != null) {
				return;
			}
			if (!alreadyHandled && subEntArea.classList.contains("editMode")) {
				// check new entry is same type with current selected entry, otherwise it cannot be expanded.
				let selEnt = document.querySelector(".subEntry.selected");
				if ((selEnt != null) && (ent.dataset.entryType != selEnt.dataset.entryType)) {
					printErrorStatus("entry type is different from selected entries");
					return;
				}
				if (!event.shiftKey || subEntArea.querySelector(".lastClicked") == null) {
					// select/deselect single entry.
					if (ent.classList.contains("selected")) {
						ent.classList.remove("selected");
					} else {
						ent.classList.add("selected");
					}
					for (let temp of subEntArea.querySelectorAll(".temporary")) {
						temp.classList.remove("temporary");
					}
					removeClass(subEntArea, "lastClicked");
					ent.classList.add("lastClicked");
				} else {
					// select/deselect multiple entries.
					let lastClicked = subEntArea.querySelector(".lastClicked");
					for (let temp of subEntArea.querySelectorAll(".temporary")) {
						temp.classList.remove("temporary");
						if (temp == lastClicked) {
							continue;
						}
						// revert to the previous status
						if (temp.classList.contains("selected")) {
							temp.classList.remove("selected");
						} else {
							temp.classList.add("selected");
						}
					}
					let range = [];
					for (let i in subEntries) {
						let e = subEntries[i];
						if (e == ent || e == lastClicked) {
							range.push(Number(i)); // wierd, but i is string
						}
					}
					if (range.length == 1) {
						range.push(range[0]);
					}
					if (range.length != 2) {
						printErrorStatus("could not find selection range");
						return;
					}
					let [from, to] = range;
					for (let i = from; i <= to; i++) {
						let e = subEntries[i];
						if (window.getComputedStyle(e).display == "none") {
							continue;
						}
						e.classList.add("temporary");
						if (lastClicked.classList.contains("selected")) {
							e.classList.add("selected");
						} else {
							e.classList.remove("selected");
						}
					}
				}
				let what = "";
				let entry = "entry"
				let selEnts = document.querySelectorAll(".subEntry.selected");
				if (selEnts.length == 0) {
					what = "no entry";
				} else if (selEnts.length == 1) {
					what = "1 entry";
				} else {
					what = String(selEnts.length) + " entries";
				}
				printStatus(what + " selected");
				if (document.querySelector(".subEntry.selected") == null) {
					hideInfoModifier();
				}
			}
		}
	}
	let statusLabelers = document.getElementsByClassName("statusLabeler");
	for (let labeler of statusLabelers) {
		let label = document.getElementById("statusLabel");
		labeler.onmouseenter = function(event) {
			let status = labeler.dataset.value;
			if (status == "") {
				status = "(none)"
			}
			label.innerText = status;
			label.style.fontSize = "0.8rem";
			label.classList.remove("nodisplay");
			let right = labeler.closest(".right");
			let offset = offsetFrom(labeler, right);
			label.style.left = String(offset.left - 4) + "px";
			label.style.top = String(offset.top - label.offsetHeight - 3) + "px";
		}
		labeler.onmouseleave = function(event) {
			label.classList.add("nodisplay");
		}
	}
	let summaryLabelers = document.getElementsByClassName("summaryLabeler");
	for (let labeler of summaryLabelers) {
		let label = document.getElementById("statusLabel");
		labeler.onmouseenter = function(event) {
			label.innerText = "";
			let assignee = labeler.dataset.assignee;
			if (assignee != "") {
				let called = CalledByName[assignee];
				label.innerText += called;
			}
			label.innerText += " / "
			let status = labeler.dataset.value;
			if (status != "") {
				// don't show '(none)' as it is too eye catch.
				label.innerText += status;
			}
			label.style.fontSize = "0.6rem";
			label.classList.remove("nodisplay");
			label.style.left = String(labeler.offsetLeft - 4) + "px";
			label.style.top = String(labeler.offsetTop - label.offsetHeight - 3) + "px";
		}
		labeler.onmouseleave = function(event) {
			label.classList.add("nodisplay");
		}
	}
	let thumbs = document.getElementsByClassName('thumbnail');
	for (let thumb of thumbs) {
		thumb.ondragover = function(event) {
			event.stopPropagation();
			event.preventDefault();
			event.currentTarget.classList.add("prepareDrop");
		}
		thumb.ondragleave = function(event) {
			event.stopPropagation();
			event.preventDefault();
			event.currentTarget.classList.remove("prepareDrop");
		}
		thumb.ondrop = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumbInput = event.currentTarget.getElementsByClassName("updateThumbnailInput")[0];
			thumbInput.files = event.dataTransfer.files;
			let thumb = parentWithClass(thumbInput, "thumbnail");
			updateThumbnail(thumb);
			event.currentTarget.classList.remove("prepareDrop");
		}
	}
	let thumbInputs = document.getElementsByClassName("updateThumbnailInput");
	for (let thumbInput of thumbInputs) {
		thumbInput.onchange = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = parentWithClass(thumbInput, "thumbnail");
			updateThumbnail(thumb);
		}
	}
	let delThumbButtons = document.getElementsByClassName("deleteThumbnailButton");
	for (let delButton of delThumbButtons) {
		delButton.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = parentWithClass(delButton, "thumbnail");
			deleteThumbnail(thumb);
		}
	}
	let assigneeInputs = document.getElementsByClassName("assigneeInput")
	for (let input of assigneeInputs) {
		let called = CalledByName[input.dataset.assignee];
		if (!called) {
			called = "";
		}
		input.value = called;
		input.dataset.oldValue = called;
		input.onfocus = function(event) {
			let editMode = subEntArea.classList.contains("editMode");
			if (!editMode) {
				input.blur();
			}
		}
		let menuAt = getOffset(input);
		menuAt.top += input.getBoundingClientRect().height + 4;
		autoComplete(input, AllUserLabels, AllUserNames, menuAt, function(value) {
			let thisEnt = parentWithClass(input, "subEntry");
			let entPath = thisEnt.dataset.entryPath;
			let selectedEnts = document.querySelectorAll(".subEntry.selected");
			if (selectedEnts.length != 0) {
				let inSel = false;
				for (let ent of selectedEnts) {
					if (entPath == ent.dataset.entryPath) {
						inSel = true;
						break;
					}
				}
				if (!inSel) {
					printErrorStatus("entry not in selection: " + entPath);
					return;
				}
			}
			if (selectedEnts.length == 0) {
				if (value == input.dataset.oldValue) {
					return;
				}
				selectedEnts = [thisEnt];
			}
			let ents = []
			for (let ent of selectedEnts) {
				ents.push(ent.dataset.entryPath);
			}
			let onsuccess = function() {
				let called = CalledByName[value];
				if (!called) {
					called = "";
				}
				input.value = called;
				input.dataset.oldValue = called;
				if (!called) {
					return;
				}
				for (let ent of selectedEnts) {
					let input = ent.getElementsByClassName("assigneeInput")[0];
					input.dataset.oldValue = called;
					input.value = called;
				}
			}
			requestPropertyUpdate(ents, "assignee", value, onsuccess);
		});
	}
	let grandSubAdderInputs = document.querySelectorAll(".grandSubAdderInput");
	for (let input of grandSubAdderInputs) {
		input.onkeydown = function() {
			if (event.key == "Enter") {
				event.preventDefault();
				let creating = input.textContent;
				let selected = document.querySelectorAll(".subEntry.selected");
				if (selected.length == 0) {
					let thisEnt = event.target.closest(".subEntry");
					selected = [thisEnt];
				}
				let formData = new FormData();
				let paths = [];
				let types = [];
				let nBypass = 0;
				for (let sel of selected) {
					if (sel.querySelector(`.grandSubEntry[data-name="${creating}"]`)) {
						// The parent already has entry we want to create.
						nBypass += 1;
						continue;
					}
					let parent = sel.dataset.entryPath;
					if (parent == "/") {
						parent = "";
					}
					let path = parent + "/" + creating;
					formData.append("path", path);
					// possibleTypes actually should just a type here.
					let type = sel.dataset.possibleSubTypes;
					formData.append("type", type);
				}
				if (nBypass == selected.length) {
					printStatus("nothing to do; all the entries already have '" + creating + "' entry");
					return;
				}
				let req = new XMLHttpRequest();
				req.open("post", "/api/add-entry");
				req.onerror = function() {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				req.onload = function() {
					if (req.status != 200) {
						printErrorStatus("cannot add entry: " + req.responseText);
						return;
					}
					location.reload();
				}
				req.send(formData);
			}
		}
	}
	let infoTitles = document.getElementsByClassName("infoTitle");
	for (let t of infoTitles) {
		t.onclick = function(event) {
			if (subEntArea.contains(t) && !subEntArea.classList.contains("editMode")) {
				return;
			}
			let info = parentWithClass(t, "info");
			showInfoUpdater(info);
		}
	}
	let infoSelectors = document.getElementsByClassName("infoSelector");
	for (let s of infoSelectors) {
		let tgl = parentWithClass(s, "infoCategoryToggle");
		s.onclick = function() {
			showCategoryInfos(tgl.dataset.category);
		}
	}
	let infoAdders = document.getElementsByClassName("infoAdder");
	for (let a of infoAdders) {
		let ent = parentWithClass(a, "entry");
		let tgl = parentWithClass(a, "infoCategoryToggle");
		a.onclick = function() {
			showInfoAdder(ent.dataset.entryPath, tgl.dataset.category);
		}
	}
	let infoContextMenuLoaders = document.getElementsByClassName("infoContextMenuLoader");
	for (let loader of infoContextMenuLoaders) {
		loader.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let ent = parentWithClass(loader, "entry");
			let info = parentWithClass(loader, "info");
			if (info == null) {
				console.log("info not found");
				return;
			}
			let menu = document.getElementById("infoContextMenu");
			if (currentContextMenuLoader == loader) {
				currentContextMenuLoader = null;
				menu.classList.add("invisible");
				return;
			}
			let infoHistory = menu.getElementsByClassName("infoHistory")[0];
			infoHistory.href = "/logs?path=" + info.dataset.entryPath + "&category=" + info.dataset.category + "&name=" + info.dataset.name;
			let infoDelete = menu.getElementsByClassName("infoDelete")[0];
			if (info.dataset.category == "property") {
				infoDelete.classList.add("nodisplay");
			}
			infoDelete.onclick = function(ev) {
				ev.preventDefault();
				let dlg = document.querySelector("#deleteInfoDialog");
				let ctg = info.dataset.category;
				dlg.querySelector(".title").innerText = "Delete " + ctg.charAt(0).toUpperCase() + ctg.substring(1);
				dlg.querySelector(".content").innerText = "Do you really want to delete '" + info.dataset.name + "' " + info.dataset.category + "?"
				let bg = document.querySelector("#deleteInfoDialogBackground");
				bg.classList.remove("invisible");
				let cancelBtn = dlg.querySelector(".cancelButton");
				cancelBtn.onclick = function() {
					bg.classList.add("invisible");
				}
				let confirmBtn = dlg.querySelector(".confirmButton");
				confirmBtn.onclick = function() {
					let req = new XMLHttpRequest();
					let formData = new FormData();
					formData.append("path", info.dataset.entryPath);
					formData.append("name", info.dataset.name);
					req.open("post", "/api/delete-" + info.dataset.category);
					req.send(formData);
					req.onload = function() {
						if (req.status == 200) {
							location.reload();
						} else {
							printErrorStatus(req.responseText);
						}
					}
					req.onerror = function(err) {
						printErrorStatus("network error occurred. please check whether the server is down.");
					}
				}
			}
			let x = loader.offsetLeft;
			let y = loader.offsetTop + loader.offsetHeight;
			menu.classList.remove("invisible");
			menu.style.left = x + "px";
			menu.style.top = y + "px";
			currentContextMenuLoader = loader;
		}
	}
	let addSubEntForms = document.querySelectorAll(".addSubEntryForm");
	for (let form of addSubEntForms) {
		form.onsubmit = function() {
			let value = form.name.value.trim();
			if (value == "") {
				printErrorStatus("nothing to submit");
				return false;
			}
			let parent = form.dataset.parent;
			if (parent == "/") {
				// prevent double slash on paths.
				parent = "";
			}
			let type = form.dataset.type;
			let req = new XMLHttpRequest();
			let formData = new FormData();
			for (let name of form.name.value.split(" ")) {
				let path = parent + "/" + name;
				formData.append("path", path);
				formData.append("type", type);
			}
			req.open("post", "/api/add-entry");
			req.onerror = function() {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status == 200) {
					location.reload(true);
				} else {
					printErrorStatus(req.responseText);
				}
			}
			req.send(formData);
			// Handled already, no need to submit again.
			return false;
		}
	}
	let infoModifiers = document.querySelectorAll(".infoModifier");
	for (let mod of infoModifiers) {
		let closeBtn = mod.querySelector(".closeButton");
		closeBtn.onclick = function() {
			mod.classList.add("nodisplay");
		}
	}
	let dots = document.querySelectorAll(".recentlyUpdatedDot");
	for (let dot of dots) {
		if (dot.classList.contains("invisible")) {
			continue;
		}
		titleRecentlyUpdatedDot(dot);
	}
}

window.onpageshow = function() {
	let thumbnailImgs = document.getElementsByClassName("thumbnailImg");
	for (let img of thumbnailImgs) {
		if ((window.getComputedStyle(img).visibility == "visible") && (img.naturalWidth == 0)) {
			// TODO: It seems the following argument isn't true anymore. Please check again if we can delete this code.
			//
			// This means we uploaded the thumbnail, but doesn't show properly
			// as we arrive this page with browser's previous button.
			// Interestingly the "?t=curent-time" part are gone (at least) in firefox.
			// It makes the browser uses the old 'not found' cache.
			img.src = img.src.split("?")[0] + "?t=" + new Date().getTime();
		}
	}
}

function removeClass(parent, clsName) {
	let elems = parent.getElementsByClassName(clsName);
	for (let e of elems) {
		e.classList.remove(clsName);
	}
}

function parentWithClass(from, clsName) {
	while (true) {
		let parent = from.parentElement;
		if (parent == null) {
			return null;
		}
		if (parent.classList.contains(clsName)) {
			return parent;
		}
		from = parent;
	}
}

function offsetFrom(elem, target) {
	if (!target) {
		return {top: 0, left: 0};
	}
	let top = elem.offsetTop;
	let left = elem.offsetLeft;
	let parent = elem.offsetParent;
	while (parent) {
		top += parent.offsetTop;
		left += parent.offsetLeft;
		if (parent == target) {
			break
		}
		parent = parent.offsetParent;
	}
	return {top: top, left: left};
}

function getOffset(el) {
  const rect = el.getBoundingClientRect();
  return {
    left: rect.left + window.scrollX,
    top: rect.top + window.scrollY
  };
}

function removeDragDropEvents(el) {
	el.ondragenter = null;
	el.ondragover = null;
	el.ondragleave = null;
	el.ondrop = null;
}

function updatePinnedPath(path, at) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("update_pinned_path", "1");
	formData.append("pinned_path", path);
	formData.append("pinned_path_at", at);
	req.open("post", "/api/update-user-setting");
	req.send(formData);
	req.onload = function() {
		if (req.status == 200) {
			location.reload();
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function updateQuickSearch(path, at) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("arrange_quick_search", "1");
	formData.append("quick_search_name", path);
	formData.append("quick_search_at", at);
	req.open("post", "/api/update-user-setting");
	req.send(formData);
	req.onload = function() {
		if (req.status == 200) {
			location.reload();
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function toggleRenameInput() {
	let input = document.getElementById("renameInput");
	if (input.classList.contains("nodisplay")) {
		input.classList.remove("nodisplay");
		let end = input.value.length;
		input.setSelectionRange(0, end);
		input.focus();
	} else {
		input.classList.add("nodisplay");
	}
}

function titleRecentlyUpdatedDot(dot) {
	let then = new Date(dot.dataset.updatedAt);
	let now = new Date();
	let today = new Date(now.toDateString());
	let dur = then - today;
	let day = 24 * 60 * 60 * 1000;
	let n = Math.floor(dur / day);
	let title = "updated today";
	if (n < 0) {
		n = Math.abs(n);
		if (n == 1) {
			title = "updated 1 day ago";
		} else {
			title = "updated " + n + " days ago";
		}
	}
	dot.title = title;
}

function updateThumbnail(thumb) {
	let img = thumb.getElementsByClassName("thumbnailImg")[0];
	let form = thumb.getElementsByClassName("updateThumbnailForm")[0];
	let now = new Date().getTime();
	if (thumb.dataset.lastUpload) {
		// Prevent Safari from firing this event twice.
		// TODO: resolve the base problem
		let last = Number(thumb.dataset.lastUpload);
		let d = now - last;
		if (d < 1000) {
			return;
		}
	}
	thumb.dataset.lastUpload = String(now);
	let req = new XMLHttpRequest();
	if (thumb.classList.contains("exists")) {
		form.action = form.action.replace("/api/add", "/api/update");
	} else {
		form.action = form.action.replace("/api/update", "/api/add");
	}
	req.open(form.method, form.action);
	req.send(new FormData(form));
	req.onload = function() {
		if (req.status == 200) {
			let entryPath = parentWithClass(thumb, "entry").dataset.entryPath;
			img.src = "/thumbnail" + entryPath + "?t=" + new Date().getTime();
			thumb.classList.remove("inherited");
			thumb.classList.add("exists");
			printStatus("done");
		} else {
			img.parentElement.style.border = "1px solid #D72";
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		img.parentElement.style.border = "1px solid #D72";
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function deleteThumbnail(thumb) {
	let img = thumb.getElementsByClassName("thumbnailImg")[0];
	let form = thumb.getElementsByClassName("deleteThumbnailForm")[0];
	let req = new XMLHttpRequest();
	req.open(form.method, form.action);
	req.send(new FormData(form));
	req.onload = function() {
		if (req.status == 200) {
			// the image is gone, reflect it to img tag (even if it will not visible).
			// TODO: inherit parent thumbnail
			img.src = img.src.split("?")[0] + "?t=" + new Date().getTime();
			form.action = form.action.replace("/api/update", "/api/add");
			thumb.classList.remove("exists");
			printStatus("done");
		} else {
			img.parentElement.style.border = "1px solid #D72";
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		img.parentElement.style.border = "1px solid #D72";
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function resizeTextArea(textarea) {
	textarea.style.height = "auto";
	textarea.style.height = String(textarea.scrollHeight) + "px";
}

function submitUpdaterOrAdder(ev, input) {
	ev.stopPropagation();
	ev.preventDefault();
	let req = new XMLHttpRequest();
	let form = input.parentElement;
	let formData = new FormData(input.parentElement);
	let entPath = formData.get("path");
	formData.delete("path"); // will be refilled
	let submitEntPaths = [];
	let thisEntry = document.querySelector(`.entry[data-entry-path="${entPath}"]`);
	if (!thisEntry || thisEntry.classList.contains("mainEntry")) {
		// thisEntry can be null if it is an inherited info.
		submitEntPaths = [entPath];
	} else {
		// subEntry
		let selectedEnts = document.querySelectorAll(".subEntry.selected");
		if (selectedEnts.length == 0) {
			submitEntPaths = [entPath];
		} else {
			for (let ent of selectedEnts) {
				submitEntPaths.push(ent.dataset.entryPath);
			}
			if (!submitEntPaths.includes(entPath)) {
				printErrorStatus("entry not in selection: " + entPath);
				return;
			}
		}
	}
	for (let path of submitEntPaths) {
		formData.append("path", path);
	}
	let ctg = formData.get("ctg");
	let name = formData.get("name");
	let marker = form.getElementsByClassName("updatingMarker")[0];
	req.onerror = function(err) {
		marker.classList.add("invisible");
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.onload = function() {
		marker.classList.add("invisible");
		if (req.status == 200) {
			// we know the value we just send,
			// but let's get the corrected value from server.
			let get = new XMLHttpRequest();
			let getFormData = new FormData();
			for (let path of submitEntPaths) {
				getFormData.append("path", path);
			}
			getFormData.append("name", name);
			get.onerror = function(err) {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			get.onload = function() {
				if (get.status == 200) {
					let j = JSON.parse(get.responseText);
					if (j.Err != "") {
						printErrorStatus(j.Err);
						return;
					}
					for (let path of submitEntPaths) {
						let infoElem = document.querySelector(`.info[data-entry-path='${path}'][data-category='${ctg}'][data-name='${name}']`);
						if (!infoElem) {
							continue
						}
						let valueElem = infoElem.querySelector(".infoValue");
						// Update the value.
						//
						// Similar code is registered as a template function in page_handler.go
						// Modify both, if needed.
						valueElem.innerHTML = "";
						let value = j.Msg.Value;
						infoElem.dataset.value = value;
						let evaled = j.Msg.Eval;
						for (let line of evaled.split("\n")) {
							line = line.trim();
							if (line == "") {
								valueElem.innerHTML += "<br>"
								continue
							}
							let div = document.createElement("div");
							let text = document.createTextNode(line);
							div.appendChild(text);
							if (line.startsWith("/")) {
								div.classList.add("pathText");
							}
							valueElem.appendChild(div);
						}
						// remove possible 'invalid' class
						valueElem.classList.remove("invalid");

						// Look UpdatedAt to check it was actually updated.
						// It might not, if new value is same as the old one.
						let updated = new Date(j.Msg.UpdatedAt);
						let now = Date.now();
						let delta = (now - updated);
						let day = 24 * 60 * 60 * 1000;
						if (delta <= day) {
							let dot = infoElem.querySelector(".recentlyUpdatedDot");
							let ent = dot.closest(".entry");
							let entDot = ent.querySelector(".recentlyUpdatedDot.forEntry");
							for (let d of [dot, entDot]) {
								d.dataset.updatedAt = j.Msg.UpdatedAt;
								d.title = "updated just now";
								d.classList.remove("invisible");
							}
						}
					}
					printStatus("done");
				} else {
					printErrorStatus("update done, but failed to get the new value:" + get.responseText);
				}
			}
			get.open("post", "/api/get-" + ctg);
			get.send(getFormData);
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.open(form.method, form.action);
	req.send(formData);
	marker.classList.remove("invisible");
}

function showCategoryInfos(ctg) {
	let cont = document.querySelector(".mainEntryInfoContainer");
	if (cont.dataset.selectedCategory == ctg) {
		ctg = ""
	}
	cont.dataset.selectedCategory = ctg;
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("update_entry_page_selected_category", "1")
	formData.append("category", ctg)
	req.open("post", "/api/update-user-setting");
	req.onerror = function() {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.onload = function() {
		if (req.status != 200) {
			printErrorStatus(req.responseText);
		}
	}
	req.send(formData);
}

function showInfoUpdater(info) {
	document.getElementById("infoAdder").classList.add("nodisplay");
	let updater = document.getElementById("infoUpdater");
	updater.classList.add("nodisplay");
	let active = document.querySelector(".infoTitle.active");
	if (active) {
		active.classList.remove("active");
	}
	let thisEnt = parentWithClass(info, "entry");
	let entPath = info.dataset.entryPath;
	let ctg = info.dataset.category;
	let name = info.dataset.name;
	let type = info.dataset.type;
	let value = info.dataset.value;
	let label = entPath;
	if (thisEnt.classList.contains("subEntry")) {
		let selectedEnts = document.querySelectorAll(".subEntry.selected");
		if (selectedEnts.length != 0) {
			let inSel = false;
			for (let ent of selectedEnts) {
				if (entPath == ent.dataset.entryPath) {
					inSel = true;
					break;
				}
			}
			if (!inSel) {
				printErrorStatus("entry not in selection: " + entPath);
				return;
			}
		}
		if (info.classList.contains("invalid")) {
			printErrorStatus(ctg + " not exists: " + name);
			return;
		}
		label = String(selectedEnts.length) + " entries selected";
		if (selectedEnts.length == 1) {
			label = entPath;
		} else if (selectedEnts.length == 0) {
			// implicit selection
			label = entPath;
		}
	}
	updater.getElementsByClassName("entryLabel")[0].innerText = label;
	updater.getElementsByClassName("entryInput")[0].value = entPath;
	updater.getElementsByClassName("categoryInput")[0].value = ctg;
	updater.getElementsByClassName("nameLabel")[0].innerText = name;
	updater.getElementsByClassName("nameInput")[0].value = name;
	updater.getElementsByClassName("typeInput")[0].value = type;
	updater.getElementsByClassName("valueForm")[0].action = "/api/update-" + ctg;
	updater.getElementsByTagName("button")[0].innerText = "Update";
	clearStatus();

	let infoTitle = info.querySelector(".infoTitle");
	infoTitle.classList.add("active");
	updater.classList.remove("nodisplay");

	let valueInput = updater.getElementsByClassName("valueInput")[0];
	valueInput.placeholder = type;
	valueInput.value = value;
	resizeTextArea(valueInput);
	valueInput.focus();
}

function hideInfoUpdater() {
	let updater = document.getElementById("infoUpdater");
	updater.classList.add("nodisplay");
}

let PropertyTypes = {{marshalJS $.PropertyTypes}}
let AccessorTypes = {{marshalJS $.AccessorTypes}}

function showInfoAdder(entry, ctg) {
	// TODO: Add the item inplace?
	document.getElementById("infoAdder").classList.remove("nodisplay");
	document.getElementById("infoUpdater").classList.add("nodisplay");

	let adder = document.getElementById("infoAdder");
	adder.classList.remove("nodisplay");
	adder.getElementsByClassName("entryLabel")[0].innerText = entry;
	adder.getElementsByClassName("entryInput")[0].value = entry;
	adder.getElementsByClassName("categoryInput")[0].value = ctg;
	adder.getElementsByTagName("button")[0].innerText = "Add";
	clearStatus();

	let nameInput = adder.getElementsByClassName("nameInput")[0]
	nameInput.value = name;
	nameInput.placeholder = ctg;
	let typeSel = adder.getElementsByClassName("typeSelect")[0]
	typeSel.innerHTML = "";
	if (ctg == "access") {
		typeSel.style.display = "none";
	} else {
		typeSel.style.removeProperty("display");
		let types = PropertyTypes;
		for (let t of types) {
			let option = document.createElement("option");
			option.value = t;
			option.text = t;
			typeSel.appendChild(option)
		}
	}
	adder.getElementsByClassName("valueForm")[0].action = "/api/add-" + ctg;

	let valueInput = adder.getElementsByClassName("valueInput")[0];
	valueInput.value = "";
	resizeTextArea(valueInput);

	nameInput.focus();
}

function hideInfoAdder() {
	let adder = document.getElementById("infoAdder");
	adder.classList.add("nodisplay");
}

let currentContextMenuLoader = null;

function printStatus(s) {
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.remove("error");
	statusBar.innerHTML = s;
}

function printErrorStatus(e) {
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.add("error");
	statusBar.innerHTML = e;
}

function clearStatus() {
	printStatus("");
}

function showInfoModifier() {
	let footer = document.querySelectorAll("infoModifier");
	if (footer.classList.contains("nodisplay")) {
		footer.classList.remove("nodisplay");
		return true;
	}
	return false;
}

function hideInfoModifier() {
	let hide = false;
	let modifiers = document.querySelectorAll(".infoModifier");
	for (let mod of modifiers) {
		if (!mod.classList.contains("nodisplay")) {
			mod.classList.add("nodisplay");
			hide = true;
		}
	}
	return hide;
}

function toggleCollapse() {
	let collapsible = this.parentElement;
	let content = collapsible.getElementsByClassName("content")[0];
	if (content.classList.contains("nodisplay")) {
		content.classList.remove("nodisplay");
	} else {
		content.classList.add("nodisplay");
	}
}

var coll = document.getElementsByClassName("collapsible");
for (let i = 0; i < coll.length; i++) {
	coll[i].getElementsByClassName("title")[0].addEventListener("click", toggleCollapse);
}

function openDeleteEntryDialog(path) {
	// The dialog itself is not hidden but the parent sets the visibility.
	let bg = document.querySelector("#deleteEntryDialogBackground");
	let dlg = document.querySelector("#deleteEntryDialog");
	let numSubEntries = -1;
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("path", path);
	req.open("post", "/api/count-all-sub-entries");
	req.send(formData);
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.onload = function() {
		if (req.status != 200) {
			printErrorStatus(req.responseText);
			return;
		}
		let j = JSON.parse(req.responseText);
		if (j.Err != "") {
			printErrorStatus(j.Err);
			return;
		}
		numSubEntries = j.Msg;
		if (numSubEntries != 0) {
			document.querySelector("#deleteEntryDialogNoSub").classList.add("nodisplay");
			document.querySelector("#deleteEntryDialogHasSub").classList.remove("nodisplay");
			document.querySelector("#deleteEntryDialogTotalSub").innerText = String(numSubEntries);
		} else {
			document.querySelector("#deleteEntryDialogNoSub").classList.remove("nodisplay");
			document.querySelector("#deleteEntryDialogHasSub").classList.add("nodisplay");
			document.querySelector("#deleteEntryDialogTotalSub").innerText = "";
		}
		document.querySelector("#deleteEntryDialogEntry").innerText = path;
		bg.classList.remove("invisible");
	}
	// cancel or confirm delete
	dlg.querySelector(".cancelButton").onclick = function() {
		bg.classList.add("invisible");
	}
	document.querySelector(".confirmButton").onclick = function() {
		let req = new XMLHttpRequest();
		let formData = new FormData();
		formData.append("path", path);
		if (numSubEntries != 0) {
			formData.append("recursive", path);
		}
		req.open("post", "/api/delete-entry");
		req.send(formData);
		req.onerror = function(err) {
			printErrorStatus("network error occurred. please check whether the server is down.");
			bg.classList.add("invisible");
		}
		req.onload = function() {
			if (req.status != 200) {
				printErrorStatus(req.responseText);
				bg.classList.add("invisible");
				return;
			}
			let toks = path.split("/");
			toks.pop();
			let parent = toks.join("/");
			if (parent == "") {
				parent = "/";
			}
			window.location.href = parent;
		}
	}
}

let AllUserNames = [
{{- range $u := $.AllUsers -}}
	"{{$u.Name}}",
{{end}}
];

let AllUserLabels = [
{{- range $u := $.AllUsers -}}
	"{{$u.Called}} ({{$u.Name}})",
{{end}}
];

// pun intended
let CalledByName = {
{{- range $u := $.AllUsers -}}
	"{{$u.Name}}": "{{$u.Called}}",
{{end}}
}

// autoComplete takes input tag and possible autocompleted values and label.
// It takes oncomplete function as an argument that will be called with user selected value.
// It will give oncomplete raw input value when it cannot find any item with the value.
function autoComplete(input, labels, vals, menuAt, oncomplete) {
	// Turn off browser's default autocomplete behavior.
	input.setAttribute("autocomplete", "off");
	let focus = -1;
	input.oninput = function(event) {
		let search = input.value;
		if (search == "") {
			return;
		}
		let lsearch = search.toLowerCase();
		// reset focus on further input.
		let focus = -1;
		let menu = document.getElementById("userAutoCompleteMenu");
		menu.classList.add("invisible");
		menu.style.left = String(menuAt.left) + "px";
		menu.style.top = String(menuAt.top) + "px";
		menu.replaceChildren();
		let items = [];
		for (let [i, l] of labels.entries()) {
			let ll = l.toLowerCase();
			let matchStart = ll.indexOf(lsearch);
			if (matchStart == -1) {
				continue
			}
			let matchEnd = matchStart + lsearch.length;
			let pre = l.slice(0, matchStart);
			let match = l.slice(matchStart, matchEnd);
			let post = l.slice(matchEnd, l.length);
			let item = document.createElement("div");
			item.classList.add("userAutoCompleteItem");
			item.innerHTML = pre + "<strong>" + match + "</strong>" + post;
			item.dataset.label = labels[i];
			item.dataset.value = vals[i];
			menu.appendChild(item);
		}
		if (menu.children.length != 0) {
			menu.classList.remove("invisible");
		}
	}
	// Don't set input.onkeydown, it will swipe default (typing characters) behavior of input.
	input.addEventListener("keydown", function(event) {
		let menu = document.getElementById("userAutoCompleteMenu");
		let items = menu.getElementsByClassName("userAutoCompleteItem");
		if (event.key == "Tab") {
			// Let the cursor move to another input.
			menu.replaceChildren();
			menu.classList.add("invisible");
			return;
		}
		deactivate(items);
		if (event.key == "ArrowDown") {
			event.preventDefault();
			focus++;
			if (focus == items.length) {
				focus = -1;
			} else {
				activate(items, focus);
			}
		} else if (event.key == "ArrowUp") {
			event.preventDefault();
			if (focus == -1) {
				focus = items.length;
			}
			focus--;
			if (focus != -1) {
				activate(items, focus);
			}
		} else if (event.key == "Enter") {
			if (items.length == 0) {
				menu.replaceChildren();
				menu.classList.add("invisible");
				focus = -1;
				return;
			}
			event.stopImmediatePropagation();
			event.preventDefault();
			if (focus == -1) {
				focus = 0;
			}
			oncomplete(items[focus].dataset.value);
			menu.replaceChildren();
			menu.classList.add("invisible");
			focus = -1;
		}
	})
	input.onkeyup = function(event) {
		let menu = document.getElementById("userAutoCompleteMenu");
		if (input.value == "") {
			menu.replaceChildren();
			menu.classList.add("invisible");
		}
	}
	function deactivate(items) {
		for (let item of items) {
			item.classList.remove("active");
		}
	}
	function activate(items, focus) {
		if (items.length == 0) {
			return;
		}
		if (focus == -1) {
			return;
		}
		items[focus].classList.add("active");
	}
}

function requestPropertyUpdate(ents, prop, value, onsuccess) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	for (let ent of ents) {
		formData.append("path", ent);
	}
	formData.append("name", "assignee");
	formData.append("ctg", "property");
	formData.append("value", value);
	req.open("post", "/api/update-property");
	req.send(formData);
	req.onerror = function() {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.onload = function() {
		if (req.status != 200) {
			printErrorStatus(req.responseText);
			return;
		}
		onsuccess()
		printStatus("done");
	}
}

function reloadPropertyPicker(popup, prop) {
	let nameInput = popup.querySelector(".propertyPickerName");
	let valueInput = popup.querySelector(".propertyPickerValue");
	nameInput.dataset.value = prop;
	if (prop == "") {
		nameInput.dataset.error = "";
		nameInput.dataset.modified = "";
		valueInput.disabled = "1";
		valueInput.value = "";
		return;
	}
	valueInput.disabled = "";
	let mainDiv = document.querySelector(".main");
	let entPath = popup.dataset.entryPath;
	let path = entPath;
	if (popup.dataset.sub != "") {
		path += "/" + popup.dataset.sub
	}
	let r = new XMLHttpRequest();
	let fdata = new FormData();
	fdata.append("path", path);
	fdata.append("name", prop);
	r.open("post", "/api/get-property");
	r.send(fdata);
	r.onerror = function() {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	r.onload = function() {
		if (r.status != 200) {
			printErrorStatus(r.responseText);
			return;
		}
		let j = JSON.parse(r.responseText);
		if (j.Err != "") {
			printErrorStatus(j.Err);
			return;
		}
		valueInput.value = j.Msg.Eval;
		nameInput.dataset.type = j.Msg.Type;
		nameInput.dataset.error = "";
		nameInput.dataset.modified = "";

		if (nameInput.dataset.type == "user") {
			let menuAt = getOffset(valueInput);
			menuAt.top += valueInput.getBoundingClientRect().height + 4;
			autoComplete(valueInput, AllUserLabels, AllUserNames, menuAt, function(value) {
				let entPath = popup.dataset.entryPath;
				let thisEnt = document.querySelector(`.subEntry[data-entry-path="${entPath}"]`)
				let selectedEnts = document.querySelectorAll(".subEntry.selected");
				if (selectedEnts.length != 0) {
					let inSel = false;
					for (let ent of selectedEnts) {
						if (entPath == ent.dataset.entryPath) {
							inSel = true;
							break;
						}
					}
					if (!inSel) {
						printErrorStatus("entry not in selection: " + entPath);
						return;
					}
				}
				if (selectedEnts.length == 0) {
					selectedEnts = [thisEnt];
				}
				let ents = []
				for (let ent of selectedEnts) {
					let path = ent.dataset.entryPath;
					if (popup.dataset.sub != "") {
						path += "/" + popup.dataset.sub;
					}
					ents.push(path);
				}
				let onsuccess = function() {
					valueInput.value = CalledByName[value];
					nameInput.dataset.error = "";
					nameInput.dataset.modified = "";
					if (nameInput.dataset.value == "assignee" && popup.dataset.sub != "") {
						for (let ent of selectedEnts) {
							let dot = ent.querySelector(`.statusSelector[data-sub="${popup.dataset.sub}"]`);
							if (!dot) {
								continue;
							}
							dot.dataset.assignee = value;
						}
					}
				}
				requestPropertyUpdate(ents, nameInput.value, value, onsuccess);
			});
		}

		printStatus("done");
	}
}
