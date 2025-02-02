"use strict";

window.onload = function() {
	document.onclick = function(event) {
		let inEditMode = document.querySelector(".subEntryArea").classList.contains("editMode");
		if (event.target.closest(".dialogBackground")) {
			return;
		}
		if (event.target.classList.contains("copyCurrentPathButton")) {
			let mainEntry = event.target.closest(".mainEntry");
			let ptxt = mainEntry.dataset.entryPath;
			let succeeded = function() {
				printStatus("entry path copied: " + ptxt);
			}
			let failed = function() {
				printStatus("failed to copy entry path");
			}
			navigator.clipboard.writeText(ptxt).then(succeeded, failed);
			return;
		}
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
		if (event.target.classList.contains("searchLink")) {
			goSearch(event.target.dataset.searchQuery);
			return;
		}
		if (event.target.classList.contains("tagLink")) {
			let t = event.target;
			let tag = t.dataset.tagName + "=" + t.dataset.tagValue;
			let path = document.querySelector("#searchArea").dataset.searchFrom;
			let url = new URL(window.location.href);
			url.pathname = path;
			let in_search = false;
			if (url.searchParams.get("search")) {
				in_search = true;
			}
			if (event.altKey || event.metaKey || !in_search) {
				url.searchParams.set("search", tag);
				window.location.href = url.toString();
				return;
			}
			let already_exists = false;
			let query = url.searchParams.get("search");
			if (url.searchParams.get("search_query")) {
				// legacy url
				query = url.searchParams.get("search_query");
			}
			for (let q of query.split(" ")) {
				if (q == tag) {
					already_exists = true;
					break;
				}
			}
			if (!already_exists) {
				query += " " + tag
			}
			url.searchParams.set("search", query)
			window.location.href = url.toString();
			return;
		}
		if (event.target.classList.contains("keyshotLink")) {
			let t = event.target;
			let query = "keyshot=" + t.dataset.entryPath;
			let path = document.querySelector("#searchArea").dataset.searchFrom;
			let url = new URL(path, window.location.origin);
			url.searchParams.set("search", query);
			window.location.href = url.toString();
			return;
		}
		if (event.target.classList.contains("assetLink")) {
			let t = event.target;
			let query = "asset=" + t.dataset.entryPath;
			let path = document.querySelector("#searchArea").dataset.searchFrom;
			let url = new URL(path, window.location.origin);
			url.searchParams.set("search", query);
			window.location.href = url.toString();
			return;
		}
		let hideBtn = event.target.closest("#sideMenuHideButton");
		if (hideBtn) {
			let left = hideBtn.closest(".left");
			let content = left.querySelector("#sideMenuContent");
			let hidden = left.classList.contains("hideSideMenu")
			if (hidden) {
				left.classList.remove("hideSideMenu");
			} else {
				left.classList.add("hideSideMenu");
			}
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("update_entry_page_hide_side_menu", "1");
			let hide = "1"
			if (hidden) {
				hide = "0"
			}
			formData.append("hide", hide);
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
		let counter = event.target.closest(".statusCounter");
		if (counter) {
			if (counter.classList.contains("sub")) {
				// not supported yet
				return;
			}
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
			let hadSelectedEntries = false;
			if (document.querySelector(".subEntry.selected")) {
				hadSelectedEntries = true;
			}
			let forTypes = document.querySelectorAll(".subEntryListForType");
			for (let forType of forTypes) {
				let typ = forType.dataset.entryType;
				for (let ent of forType.querySelectorAll(".subEntry")) {
					if (sum.dataset.selected != "1") {
						ent.classList.remove("hidden");
						continue;
					}
					if (forType.dataset.entryType != entType) {
						ent.classList.add("hidden");
						ent.classList.remove("selected");
						continue;
					}
					let dot = ent.querySelector(".statusDot");
					if (dot) {
						if (dot.dataset.value != sum.dataset.selectedStatus) {
							ent.classList.add("hidden");
							ent.classList.remove("selected");
							continue;
						}
					} else {
						// Should work for entries of type that doesn't have status.
						if (sum.dataset.selectedStatus != "") {
							ent.classList.add("hidden");
							ent.classList.remove("selected");
							continue;
						}
					}
					ent.classList.remove("hidden");
				}
				// also handle containers when user set 'group by' option.
				let nTotal = 0;
				for (let cnt of forType.querySelectorAll(".subEntryListContainer")) {
					let n = 0;
					for (let ent of cnt.querySelectorAll(".subEntry")) {
						if (!ent.classList.contains("hidden")) {
							n++;
							nTotal++;
						}
					}
					let count = cnt.querySelector(".subEntryListFromCount");
					if (count) {
						count.innerText = "(" + String(n) + ")";
					}
					if (n == 0) {
						cnt.classList.add("hidden");
					} else {
						cnt.classList.remove("hidden");
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
			if (hadSelectedEntries) {
				printSelectionStatus();
			}
			return;
		}
		let options = event.target.closest(".subEntryListOptions");
		if (options) {
			let opt = event.target.closest(".subEntryListOption.viewOption");
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
			opt = event.target.closest(".subEntryListOption.editModeOption");
			if (opt) {
				let subEntArea = document.querySelector(".subEntryArea");
				if (subEntArea.classList.contains("editMode")) {
					subEntArea.classList.remove("editMode");
					removeClass(subEntArea, "lastClicked");
					removeClass(subEntArea, "temporary");
					removeClass(subEntArea, "selected");
					printStatus("normal mode");
				} else {
					subEntArea.classList.add("editMode");
					printStatus("edit mode");
				}
				return;
			}
			opt = event.target.closest(".subEntryListOption.expandPropertyOption");
			if (opt) {
				let subEntArea = document.querySelector(".subEntryArea");
				if (opt.dataset.expand == "true") {
					subEntArea.classList.remove("expandProperty");
					opt.dataset.expand = "false";
				} else {
					subEntArea.classList.add("expandProperty");
					opt.dataset.expand = "true";
				}
				let req = new XMLHttpRequest();
				let formData = new FormData();
				formData.append("update_entry_page_expand_property", "1");
				formData.append("expand", opt.dataset.expand);
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
			opt = event.target.closest(".subEntryListOption.detailViewOption");
			if (opt) {
				let subEntArea = document.querySelector(".subEntryArea");
				if (opt.dataset.value) {
					subEntArea.classList.remove("detailView");
					opt.dataset.value = "";
				} else {
					subEntArea.classList.add("detailView");
					opt.dataset.value = "1";
				}
				let req = new XMLHttpRequest();
				let formData = new FormData();
				formData.append("section", "entry_page");
				req.open("post", "/api/ensure-user-data-section");
				req.onerror = function() {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				req.onload = function() {
					if (req.status != 200) {
						printErrorStatus(req.responseText);
						return;
					}
					let r = new XMLHttpRequest();
					let data = new FormData();
					data.append("section", "entry_page");
					data.append("key", "detail_view");
					data.append("value", opt.dataset.value);
					r.open("post", "/api/set-user-data");
					r.onerror = function() {
						printErrorStatus("network error occurred. please check whether the server is down.");
					}
					r.onload = function() {
						if (r.status != 200) {
							printErrorStatus(req.responseText);
							return;
						}
					}
					r.send(data);
				}
				req.send(formData);
				return;
			}
			opt = event.target.closest(".subEntryListOption.deleteEntryOption");
			if (opt) {
				let selEnts = document.querySelectorAll(".subEntry.selected");
				if (selEnts.length == 0) {
					printErrorStatus("no sub-entry selected");
					return;
				}
				let paths = [];
				for (let ent of selEnts) {
					paths.push(ent.dataset.entryPath);
				}
				openDeleteEntryDialog(paths);
				return;
			}
		}
		let show_hidden = event.target.closest(".showHiddenProperty");
		if (show_hidden) {
			let bottom = document.querySelector(".mainEntryBottom");
			if (bottom.dataset.showHidden == "") {
				bottom.dataset.showHidden = "1";
			} else {
				bottom.dataset.showHidden = "";
			}
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("update_entry_page_show_hidden_property", "1");
			formData.append("show_hidden", bottom.dataset.showHidden);
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
		let expander = event.target.closest(".thumbnailViewExpander");
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
			let hide = cont.dataset.hide;
			for (let c of conts) {
				if (!hide) {
					c.dataset.hide = "1"
				} else {
					c.dataset.hide = ""
				}
			}
		}
		let hide = false;
		let handle = event.target.closest(".statusSelector, #updatePropertyPopup");
		if (handle != null) {
			hide = true;
			let mainDiv = document.querySelector(".main");
			let fn = function() {
				if (handle.classList.contains("statusSelector")) {
					// open or close updatePropertyPopup
					let sel = handle;
					let thisEnt = sel.closest(".entry");
					let entPath = thisEnt.dataset.entryPath;
					let entType = sel.dataset.entryType;
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
								let popup = document.querySelector("#updatePropertyPopup");
								popup.classList.remove("expose");
								printErrorStatus("entry not in selection: " + entPath);
								return;
							}
						}
					}
					let popup = document.querySelector("#updatePropertyPopup");
					if (popup.dataset.entryPath == entPath && popup.dataset.sub == sel.dataset.sub) {
						// popup is already opened, close
						if (popup.classList.contains("expose")) {
							popup.classList.remove("expose");
							hide = true;
							return;
						}
					} else {
						// recalcuate popup
						popup.dataset.entryPath = entPath;
						popup.dataset.sub = sel.dataset.sub;
						// reset inner elements
						popup.dataset.entryType = sel.dataset.entryType;
						let menu = popup.querySelector(".selectStatusMenu");
						menu.innerHTML = "";
						let stats = PossibleStatus[sel.dataset.entryType];
						if (stats.length != 0) {
							menu.classList.remove("hidden");
							for (let s of stats) {
								let item = document.createElement("div");
								item.dataset.value = s;
								item.classList.add("selectStatusMenuItem");
								let dot = document.createElement("div");
								dot.classList.add("selectStatusMenuItemDot");
								dot.classList.add("statusDot");
								dot.dataset.entryType = sel.dataset.entryType;
								dot.dataset.value = s;
								let val = document.createElement("div");
								val.classList.add("selectStatusMenuItemValue");
								let t = s;
								if (s == "") {
									t = "(none)";
									val.style.color = "#888888";
								}
								val.innerText = t;
								item.appendChild(dot);
								item.appendChild(val);
								menu.appendChild(item);
							}
						} else {
							menu.classList.add("hidden");
						}
						let select = popup.querySelector(".propertyPickerName");
						select.innerHTML = "";
						let props = Properties[sel.dataset.entryType].slice();
						props.push("*environ", "*access");
						let picked = LastPickedProperty[sel.dataset.entryType];
						for (let p of props) {
							let opt = document.createElement("option");
							opt.value = p;
							let t = p || ">";
							opt.innerText = t;
							if (p == picked) {
								opt.selected = true;
							}
							select.appendChild(opt);
						}
						let nameInput = popup.querySelector(".propertyPickerName");
						reloadPropertyPicker(popup, nameInput.value.trim());
					}
					// slight adjust of the popup position to make statusDots aligned.
					popup.style.removeProperty("right");
					let right = sel.closest(".right");
					let offset = offsetFrom(sel, right);
					let status = popup.querySelector(".selectStatusMenu");
					let picker = popup.querySelector(".propertyPicker");
					popup.style.left = String(offset.left - 6) + "px";
					popup.style.top = String(offset.top + sel.offsetHeight + 4) + "px";
					popup.insertBefore(status, picker); // default style
					popup.classList.add("expose");
					// some times popup placed outside of window. prevent it.
					// but only when the popup fits in the window by switching positions.
					if (popup.getBoundingClientRect().right > document.body.getBoundingClientRect().right) {
						if (status.getBoundingClientRect().x - picker.getBoundingClientRect().width > 0) {
							popup.style.removeProperty("left");
							popup.style.right = "0px";
							let margin = popup.getBoundingClientRect().right - sel.getBoundingClientRect().left;
							popup.style.right = String(margin - 125) + "px";
						}
					}
					if (popup.style.right) {
						popup.insertBefore(picker, status);
					} else {
						popup.insertBefore(status, picker);
					}
				} else {
					// an element inside of #updatePropertyPopup clicked
					let popup = handle;
					let thisEnt = document.querySelector(`.entry[data-entry-path="${popup.dataset.entryPath}"]`);
					let entPath = popup.dataset.entryPath;
					let item = event.target.closest(".selectStatusMenuItem");
					if (item != null) {
						// change status when user clicked .selectStatusMenuItem
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
								popup.classList.remove("expose");
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
								let popup = document.querySelector("#updatePropertyPopup");
								popup.classList.remove("expose");
							} else {
								printErrorStatus(req.responseText);
							}
						}
						req.onerror = function(err) {
							printErrorStatus("network error occurred. please check whether the server is down.");
						}
					}
					let history = event.target.closest(".propertyPickerHistory");
					if (history != null) {
						let path = popup.dataset.entryPath;
						if (popup.dataset.sub) {
							path += "/" + popup.dataset.sub;
						}
						let prop = history.closest(".propertyPicker").querySelector(".propertyPickerName").dataset.value;
						window.location.href = "/logs?path=" + path + "&category=property&name=" + prop;
					}
				}
			}
			fn()
		} else {
			let popup = document.querySelector("#updatePropertyPopup");
			if (popup.classList.contains("expose")) {
				popup.classList.remove("expose");
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
		let dueLabel = event.target.closest(".dueLabel");
		if (dueLabel != null && !(event.altKey || event.metaKey) && !inEditMode) {
			let due = dueLabel.dataset.due;
			let ent = dueLabel.closest(".subEntry");
			let gs = dueLabel.closest(".grandSub");
			if (gs) {
				ent = gs.querySelector(".grandSubEntry");
			}
			let entType = ent.dataset.entryType;
			goSearch("type=" + entType + " due=" + due);
			hide = true;
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
			let grandSubAdding = document.querySelector(".grandSubArea.adding");
			if (grandSubAdding != null) {
				grandSubAdding.classList.remove("adding");
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
			let grandSubAdding = document.querySelector(".grandSubArea.adding");
			if (grandSubAdding != null) {
				let editable = grandSubAdding.querySelector(".grandSubAdderInput");
				editable.textContent = "";
				grandSubAdding.classList.remove("adding");
				hide = true;
			}
		}
		if (hide) {
			return;
		}
		if (event.target.closest("#searchInput, #downloadAsExcelButton, .subEntry, #footer") == null) {
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
		let scrollToTop = event.target.closest("#scrollToTop")
		if (scrollToTop) {
			window.scrollTo(window.scrollX, 0);
			scrollToTop.classList.remove("reveal");
		}
	}
	document.onkeydown = function(event) {
		if (event.repeat) {
			return;
		}
		let ctrlPressed = event.ctrlKey || event.metaKey;
		if (event.code == "Escape") {
			// Will close floating UIs first, if any exists.
			let bg = document.querySelector(".dialogBackground");
			if (!bg.classList.contains("invisible")) {
				bg.classList.add("invisible");
				return;
			}
			let hide = false;
			let popup = document.querySelector("#updatePropertyPopup");
			if (popup.classList.contains("expose")) {
				popup.classList.remove("expose");
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
			let grandSubAdding = document.querySelector(".grandSubArea.adding");
			if (grandSubAdding != null) {
				let editable = grandSubAdding.querySelector(".grandSubAdderInput");
				editable.textContent = "";
				grandSubAdding.classList.remove("adding");
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
				let popup = event.target.closest("#updatePropertyPopup");
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
				if (selEnts.length == 0) {
					let thisEnt = document.querySelector(`.entry[data-entry-path="${popup.dataset.entryPath}"]`);
					selEnts = [thisEnt];
				}

				let paths = [];
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
				if (prop == "*environ" || prop == "*access") {
					let updateType = prop.slice(1);
					let lines = valueInput.value.split("\n");
					let modify = false;
					for (let l of lines) {
						l = l.trim();
						let prefix = l.slice(0, 1);
						if (prefix != "+" && prefix != "-") {
							continue
						}
						modify = true;

						let keyVal = l.slice(1);
						keyVal = keyVal.trim();
						let idx = keyVal.indexOf("=");
						if (idx < 0) {
							printErrorStatus("unexpected line: "  + l);
							break;
						}
						let key = keyVal.slice(0, idx).trim();
						let val = keyVal.slice(idx+1).trim();

						let req = new XMLHttpRequest();
						let formData = new FormData();
						for (let path of paths) {
							formData.append("path", path);
						}
						formData.append("name", key);
						let api = "";
						if (prefix == "+") {
							api = "/api/add-or-update-" + updateType;
							formData.append("value", val);
						} else if (prefix == "-") {
							api = "/api/delete-" + updateType;
							formData.append("generous", "1");
						}
						req.open("post", api);
						req.send(formData);
						req.onerror = function() {
							nameInput.dataset.error = "1";
							printErrorStatus("network error occurred. please check whether the server is down.");
						}
						req.onload = function() {
							if (req.status != 200) {
								nameInput.dataset.error = "1";
								printErrorStatus(req.responseText);
								console.log(req.responseText);
								return;
							}
							nameInput.dataset.error = "";
							nameInput.dataset.modified = "";
							reloadPropertyPicker(popup, prop);
							printStatus("done");
						}
					}
					if (!modify) {
						printStatus("nothing to do");
					}
					return;
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
		if (subEntArea.classList.contains("editMode")) {
			// key binding for edit mode
			let userEditables = ["TEXTAREA", "INPUT"];
			if (userEditables.includes(event.target.tagName)) {
				// even in edit mode, text editing in user-editables shouldn't be interupted.
				return;
			}
			if (ctrlPressed && event.code == "KeyA") {
				event.preventDefault();
				let selEnt = document.querySelector(".subEntry.selected");
				if (!selEnt) {
					let entTypes = document.querySelectorAll(".subEntryListForType");
					if (entTypes.length > 1) {
						printErrorStatus("please select at least one entry to determine entry type");
						return;
					}
					let first = document.querySelector(".subEntry:not(.hidden)");
					if (!first) {
						return;
					}
					selEnt = first;
				}
				let typeList = selEnt.closest(".subEntryListForType");
				for (let group of typeList.querySelectorAll(".subEntryListContainer")) {
					if (group.dataset.hide) {
						continue;
					}
					let ents = group.querySelectorAll(".subEntry:not(.hidden)");
					for (let ent of ents) {
						ent.classList.add("selected");
					}
				}
				removeClass(subEntArea, "lastClicked");
				removeClass(subEntArea, "temporary");
				printSelectionStatus();
				return;
			}
			if (ctrlPressed && event.code == "KeyI") {
				event.preventDefault();
				let selEnt = document.querySelector(".subEntry.selected");
				if (!selEnt) {
					let first = document.querySelector(".subEntry:not(.hidden)");
					if (!first) {
						return;
					}
					selEnt = first;
				}
				let typ = selEnt.dataset.entryType;
				let typeEnts = document.querySelectorAll(`.subEntry:not(.hidden)[data-entry-type="${typ}"]`);
				for (let ent of typeEnts) {
					if (ent.classList.contains("selected")) {
						ent.classList.remove("selected");
					} else {
						ent.classList.add("selected");
					}
				}
				removeClass(subEntArea, "lastClicked");
				removeClass(subEntArea, "temporary");
				let firstSel = document.querySelector(".subEntry.selected");
				if (firstSel) {
					firstSel.classList.add("lastClicked");
				}
				printSelectionStatus();
				return;
			}
		}
		if (ctrlPressed && event.code == "KeyC") {
			if (["INPUT", "TEXTAREA"].includes(event.target.nodeName)) {
				return;
			}
			if (getSelection().type == "Range") {
				// user has selected some text. don't copy anything else.
				return;
			}
			// need at least one entry selected or hovered
			let copyable = null;
			let copyables = document.querySelectorAll(".copyable:hover");
			if (copyables.length != 0) {
				copyable = copyables[copyables.length-1];
			}
			if (!copyable) {
				let selected = document.querySelector(".subEntry.selected");
				if (!selected) {
					return;
				}
				copyable = selected;
			}

			event.preventDefault();

			let popup = document.querySelector("#updatePropertyPopup");
			let namePicker = popup.querySelector(".propertyPickerName");
			if (copyable == namePicker) {
				// propertyPickerName is .copyable, but has special logic for itself.
				// TODO: need to generalize the logic.
				let entPath = popup.dataset.entryPath;
				let prop = namePicker.dataset.value;
				if (prop == "") {
					return;
				}
				if (prop.startsWith("*")) {
					printStatus("not support copy: " + prop);
					return;
				}
				let thisEnt = document.querySelector(`.subEntry[data-entry-path="${entPath}"]`);
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
				let sub = popup.dataset.sub;
				let paths = []
				for (let ent of selectedEnts) {
					let path = ent.dataset.entryPath;
					if (sub != "") {
						if (ent.querySelector(`.grandSubEntry[data-sub="${sub}"]`) == null) {
							continue
						}
						path += "/" + sub;
					}
					paths.push(path);
				}
				getProperties(paths, prop, function(ps) {
					let data = "";
					for (let p of ps) {
						data += p.Eval
						data += "\n"
					}
					let succeeded = function() {
						let show = data;
						if (show.length > 50) {
							show = show.slice(0, 50) + "...";
						}
						printStatus(prop + " copied from " + ps.length + " entries: " + show);
						copyable.classList.add("highlight");
						setTimeout(function() {
							copyable.classList.remove("highlight");
						}, 500)
					}
					let failed = function() {
						printStatus("failed to copy data");
					}
					navigator.clipboard.writeText(data).then(succeeded, failed);
					return;
				});
				return;
			}

			let subEnt = copyable.closest(".subEntry");
			if (!subEnt || !copyable.dataset.copyKey) {
				let field = copyable.dataset.copyField;
				let data = copyable.dataset[field];
				let show = data;
				if (show.length > 50) {
					show = show.slice(0, 50) + "...";
				}
				let copyKey = "data";
				if (copyable.dataset.copyKey) {
					copyKey = copyable.dataset.copyKey;
				}
				let succeeded = function() {
					printStatus(copyKey + " copied: " + show);
					copyable.classList.add("highlight");
					setTimeout(function() {
						copyable.classList.remove("highlight");
					}, 500)
				}
				let failed = function() {
					printStatus("failed to copy data");
				}
				navigator.clipboard.writeText(data).then(succeeded, failed);
				return;
			}

			let selEnts = document.querySelectorAll(".subEntry.selected");
			if (selEnts.length == 0) {
				selEnts = document.querySelectorAll(".subEntry:hover");
			}
			// multiple entries selected
			let copyKey = copyable.dataset.copyKey;
			let data = "";
			let nCopy = 0;
			for (let i = 0; i < selEnts.length; i++) {
				let ent = selEnts[i];
				let c = null;
				if (ent.dataset.copyKey == copyKey) {
					// querySelector does not work for self.
					c = ent;
				} else {
					c = ent.querySelector(`.copyable[data-copy-key="${copyKey}"]`);
				}
				if (!c) {
					continue;
				}
				let field = c.dataset.copyField;
				if (c.dataset.copyFrom) {
					// it might want to get data from a parent
					c = c.closest(c.dataset.copyFrom);
				}
				nCopy += 1;
				let d = c.dataset[field];
				if (i != 0) {
					data += "\n";
				}
				data += d;
			}
			let succeeded = function() {
				let show = data;
				if (show.length > 50) {
					show = show.slice(0, 50) + "...";
				}
				let num = String(selEnts.length);
				if (nCopy != selEnts.length) {
					num = String(nCopy) + " of " + String(selEnts.length);
				}
				printStatus(copyKey + " copied from " + num + " entries: " + show);
				copyable.classList.add("highlight");
				setTimeout(function() {
					copyable.classList.remove("highlight");
				}, 500)
			}
			let failed = function() {
				printStatus("failed to copy data");
			}
			navigator.clipboard.writeText(data).then(succeeded, failed);
			return;
		}
		if (ctrlPressed && event.code == "KeyD") {
			if (["INPUT", "TEXTAREA"].includes(event.target.nodeName)) {
				return;
			}
			let ents = document.querySelectorAll(":is([data-entry-path],[data-sub]):hover");
			if (ents.length == 0) {
				return;
			}
			event.preventDefault();
			let ent = ents[ents.length-1];
			let sub = "";
			let path = "";
			if (ent.dataset.sub) {
				sub = ent.dataset.sub;
				let parent = ent.parentElement.closest(".entry");
				if (!parent) {
					return;
				}
				path = parent.dataset.entryPath + "/" + sub;
			} else {
				path = ent.dataset.entryPath;
			}
			location.href = path;
			return;
		}
	}
	document.onchange = function(event) {
		if (event.target.closest(".propertyPickerName")) {
			let popup = event.target.closest("#updatePropertyPopup");
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
			let popup = event.target.closest("#updatePropertyPopup");
			let nameInput = popup.querySelector(".propertyPickerName");
			nameInput.dataset.error = "";
			nameInput.dataset.modified = "1";
		}
	}
	let searchTypeSelect = document.querySelector("#searchTypeSelect");
	searchTypeSelect.onchange = function() {
		searchTypeSelect.classList.remove("notEffected");
	}
	let searchForm = document.querySelector("#searchForm");
	searchForm.onsubmit = function(event) {
		// it will be handled by searchInput.onkeydown;
		event.preventDefault();
		return false;
	}
	let searchInput = document.querySelector("#searchInput");
	searchInput.onkeydown = function(event) {
		if (event.code != "Enter" && event.code != "NumpadEnter") {
			return;
		}
		let formData = new FormData(searchForm);
		let query = formData.get("search");
		if (event.ctrlKey || event.metaKey) {
			// search by entry path mode
			let toks = [];
			for (let tok of query.split(" ")) {
				tok = tok.trim();
				if (tok == "/") {
					// don't contain root entry
					continue;
				}
				if (tok.startsWith("/")) {
					toks.push(tok);
				}
			}
			let newQuery = "-mode:entry";
			if (toks.length != 0) {
				newQuery += " " + toks.join(" ")
			}
			query = newQuery;
		}
		goSearch(query);
		return;
	}
	let searchButton = document.querySelector("#searchButton");
	searchButton.onclick = function(event) {
		let formData = new FormData(searchForm);
		let query = formData.get("search");
		goSearch(query);
		return;
	}
	let addQuickSearchForm = document.querySelector("#addQuickSearchForm");
	addQuickSearchForm.onsubmit = function(event) {
		let searchFormData = new FormData(searchForm);
		let searchFormParam = new URLSearchParams(searchFormData);
		let query = searchFormParam.get("search");
		let typeInQuery = false;
		for (let tok of query.split(" ")) {
			tok = tok.trim();
			if (tok.startsWith("type=")) {
				typeInQuery = true;
			}
		}
		if (typeInQuery || !searchFormParam.get("search_entry_type")) {
			searchFormParam.delete("search_entry_type");
		}
		let req = new XMLHttpRequest();
		let formData = new FormData(addQuickSearchForm);
		formData.set("update_quick_search", "1");
		formData.set("quick_search_value", searchFormParam.toString());
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
		return false;
	}
	let allInputs = document.getElementsByTagName("input");
	for (let input of allInputs) {
		input.autocomplete = "off";
	}
	let inputs = document.getElementsByClassName("valueInput");
	for (let input of inputs) {
		input.onkeydown = function(ev) {
			if (ev.repeat) {
				return;
			}
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
	let addExcelFile = document.getElementById("addExcelFile");
	if (addExcelFile != null) {
		addExcelFile.onchange = function() {
			let addExcelForm = document.getElementById("addExcelForm");
			submitForm(addExcelForm);
		}
	}
	let updateExcelFile = document.getElementById("updateExcelFile");
	if (updateExcelFile != null) {
		updateExcelFile.onchange = function() {
			let updateExcelForm = document.getElementById("updateExcelForm");
			submitForm(updateExcelForm);
		}
	}
	let downloadAsExcelButton = document.getElementById("downloadAsExcelButton");
	if (downloadAsExcelButton != null) {
		downloadAsExcelButton.onclick = function() {
			let editMode = false;
			let subEntArea = document.querySelector(".subEntryArea");
			if (subEntArea.classList.contains("editMode")) {
				editMode = true;
			}
			let ents = document.querySelectorAll(".subEntry");
			if (ents.length == 0) {
				printErrorStatus("no sub-entry exists");
				return;
			}
			let paths = [];
			for (let ent of ents) {
				if (window.getComputedStyle(ent).display == "none") {
					// invisible entry shouldn't be exported.
					continue
				}
				if (editMode && !ent.classList.contains("selected")) {
					continue;
				}
				paths.push(ent.dataset.entryPath);
			}
			if (paths.length == 0) {
				printErrorStatus("no sub-entry selected");
				return;
			}
			let formData = new FormData();
			for (let path of paths) {
				formData.append("paths", path);
			}
			let req = new XMLHttpRequest();
			req.responseType = "blob";
			req.open("post", "/download-as-excel");
			req.send(formData);
			req.onload = function() {
				if (req.status != 200) {
					let r = new FileReader();
					r.onload = function() {
						printErrorStatus(r.result);
					}
					r.readAsText(req.response);
					return;
				}
				let disposition = req.getResponseHeader('Content-Disposition');
				if (!disposition) {
					printErrorStatus("reponse does not contain excel file");
					return;
				}
				if (disposition.indexOf("attachment") == -1) {
					printErrorStatus("reponse does not contain excel file");
					return;
				}
				let downloadURL = window.URL.createObjectURL(req.response);
				let dateString = function(d) {
					function pad(n) {
						if (n < 10) {
							return "0" + n.toString();
						}
						return n.toString();
					}
					let ymd = [d.getFullYear(), pad(d.getMonth()+1), pad(d.getDate())].join("-");
					let hms = [pad(d.getHours()), pad(d.getMinutes()), pad(d.getSeconds())].join("-");
					let date = ymd + "T" + hms;
					return date;
				}
				let d = new Date();
				let a = document.createElement("a");
				a.href = downloadURL;
				let suffix = ""
				if (editMode) {
					suffix = "-selected"
				}
				a.download = "forge-" + dateString(d) + suffix + ".xlsx";
				a.click();
				setTimeout(function() {
					URL.revokeObjectURL(downloadURL);
				}, 100)
			}
			req.onerror = function(err) {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
		}
	}
	let backupAsExcelButton = document.getElementById("backupAsExcelButton");
	if (backupAsExcelButton != null) {
		backupAsExcelButton.onclick = function() {
			let mainEntry = document.querySelector(".mainEntry");
			if (!mainEntry) {
				console.log("no directory entry exists to archive")
				return;
			}
			let formData = new FormData();
			formData.append("root", mainEntry.dataset.entryPath);
			let req = new XMLHttpRequest();
			req.responseType = "blob";
			req.open("post", "/backup-as-excel");
			req.send(formData);
			req.onload = function() {
				if (req.status != 200) {
					let r = new FileReader();
					r.onload = function() {
						printErrorStatus(r.result);
					}
					r.readAsText(req.response);
					return;
				}
				let disposition = req.getResponseHeader('Content-Disposition');
				if (!disposition) {
					printErrorStatus("reponse does not contain excel file");
					return;
				}
				if (disposition.indexOf("attachment") == -1) {
					printErrorStatus("reponse does not contain excel file");
					return;
				}
				let downloadURL = window.URL.createObjectURL(req.response);
				let dateString = function(d) {
					function pad(n) {
						if (n < 10) {
							return "0" + n.toString();
						}
						return n.toString();
					}
					let ymd = [d.getFullYear(), pad(d.getMonth()+1), pad(d.getDate())].join("-");
					let hms = [pad(d.getHours()), pad(d.getMinutes()), pad(d.getSeconds())].join("-");
					let date = ymd + "T" + hms;
					return date;
				}
				let d = new Date();
				let a = document.createElement("a");
				a.href = downloadURL;
				a.download = "forge-" + dateString(d) + ".xlsx";
				a.click();
				setTimeout(function() {
					URL.revokeObjectURL(downloadURL);
				}, 100)
			}
			req.onerror = function(err) {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
		}
	}
	let pinnedPaths = document.getElementsByClassName("pinnedPathLink");
	for (let pp of pinnedPaths) {
		pp.onclick = function(event) {
			window.location.href = pp.dataset.path;
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
				updatePinnedPath(pp.dataset.path, at);
			}
			let del = document.getElementById("pinnedPathDeleteButton");
			del.classList.remove("nodisplay");
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.classList.add("prepareDrop");
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.classList.add("prepareDrop");
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.classList.remove("prepareDrop");
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				// updatePinnedPath reloads the page.
				updatePinnedPath(pp.dataset.path, -1);
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
			// for historical reason, the data encoded as uri string
			let p = new URLSearchParams(qs.dataset.search);
			let query = "";
			if (p.get("search_entry_type")) {
				query += "type=" + p.get("search_entry_type");
			}
			if (p.get("search_query")) {
				if (query) {
					query += " ";
				}
				query += p.get("search_query");
			}
			if (p.get("search")) {
				if (query) {
					query += " ";
				}
				query += p.get("search");
			}
			goSearch(query);
			return;
		}
		qs.ondragstart = function(event) {
			// I've had hard time when I drag quickSearchLink while it is 'a' tag.
			// At first glance qs.ondragstart seemed to work consitently, then the link is clicked instead.
			// Hope I got peace by making quickSearchLink 'div'.
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
				updateQuickSearch(qs.dataset.key, at, false);
			}
			let del = document.getElementById("quickSearchDeleteButton");
			del.classList.remove("nodisplay");
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.classList.add("prepareDrop");
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.classList.add("prepareDrop");
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.classList.remove("prepareDrop");
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				// updateQuickSearch reloads the page.
				updateQuickSearch(qs.dataset.key, -1, false);
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
			for (let cls of ["searchLink", "tagLink", "entryLink"]) {
				if (event.target.classList.contains(cls)) {
					// it's moving to another page
					return;
				}
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
			if (event.target.tagName == "A") {
				// it will go to other page. keep this state.
				return;
			}
			if (
				document.querySelector("#updatePropertyPopup.expose") ||
				document.querySelector("#infoUpdater:not(.nodisplay)") ||
				document.querySelector("#infoAdder:not(.nodisplay)") ||
				document.querySelector(".grandSubArea.adding")
			) {
				// close those first.
				// a bit clunky where most logic is on document.onclick,
				// but entry selection is separate from that.
				return;
			}
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
					let subEnts = document.getElementsByClassName("subEntry");
					for (let i in subEnts) {
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
						let e = subEnts[i];
						if (e.classList.contains("hidden")) {
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
				printSelectionStatus();
				if (document.querySelector(".subEntry.selected") == null) {
					hideInfoModifier();
				}
			}
		}
		let popup = document.querySelector("#updatePropertyPopup");
		popup.onmouseup = function(event) {
			let pickedPropertyInput = document.querySelector(".propertyPickerValue");
			if (pickedPropertyInput.dataset.resized != "") {
				pickedPropertyInput.dataset.resized = "";
				let width = pickedPropertyInput.style.width;
				let height = pickedPropertyInput.style.height;
				if (pickedPropertyInput.dataset.oldWidth != width || pickedPropertyInput.dataset.oldHeight != height) {
					pickedPropertyInput.dataset.oldWidth = width;
					pickedPropertyInput.dataset.oldHeight = height;
					let w = width.slice(0, -2);
					let h = height.slice(0, -2);
					let size = w + "x" + h;
					let req = new XMLHttpRequest();
					let formData = new FormData();
					formData.append("update_picked_property_input_size", size);
					formData.append("size", size);
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
			let due = labeler.dataset.due;
			if (due != "") {
				label.innerText += " / "
				let t = dday(due);
				label.innerText += t;
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
			let thumb = thumbInput.closest(".thumbnail");
			updateThumbnail(thumb);
			event.currentTarget.classList.remove("prepareDrop");
		}
	}
	let thumbInputs = document.getElementsByClassName("updateThumbnailInput");
	for (let thumbInput of thumbInputs) {
		thumbInput.onchange = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = thumbInput.closest(".thumbnail");
			updateThumbnail(thumb);
		}
	}
	let delThumbButtons = document.getElementsByClassName("deleteThumbnailButton");
	for (let delButton of delThumbButtons) {
		delButton.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = delButton.closest(".thumbnail");
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
			let thisEnt = input.closest(".subEntry");
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
			let paths = []
			for (let ent of selectedEnts) {
				paths.push(ent.dataset.entryPath);
			}
			let onsuccess = function() {
				let called = CalledByName[value];
				for (let ent of selectedEnts) {
					let inp = ent.getElementsByClassName("assigneeInput")[0];
					inp.dataset.oldValue = called;
					inp.value = called;
				}
			}
			updateProperty(paths, "assignee", value, onsuccess);
		});
	}
	let grandSubAdderInputs = document.querySelectorAll(".grandSubAdderInput");
	for (let input of grandSubAdderInputs) {
		input.onkeydown = function() {
			if (event.key == "Enter") {
				event.preventDefault();
				let thisEnt = event.target.closest(".subEntry");
				let sub = input.textContent;
				if (sub.includes(" ")) {
					printErrorStatus("cannot create entry with name that has space: " + sub);
					return;
				}
				let selected = document.querySelectorAll(".subEntry.selected");
				if (selected.length == 0) {
					selected = [thisEnt];
				}
				let paths = [];
				let types = [];
				for (let sel of selected) {
					if (sel.querySelector(`.grandSubEntry[data-sub="${sub}"]`)) {
						// The parent already has entry we want to create.
						continue;
					}
					let parent = sel.dataset.entryPath;
					if (parent == "/") {
						parent = "";
					}
					paths.push(parent + "/" + sub);
					// BUG: 'fx/a' entry should follow possibleSubTypes of 'fx', not selected entry
					// find real types for grand sub entries
					types.push(sel.dataset.possibleSubTypes);
				}
				if (paths.length == 0) {
					printStatus("nothing to do; all the entries already have '" + sub + "' entry");
					return;
				}
				let formData = new FormData();
				for (let i in paths) {
					formData.append("path", paths[i]);
					formData.append("type", types[i]);
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
					let r = new XMLHttpRequest();
					let fdata = new FormData();
					for (let path of paths) {
						let toks = path.split("/");
						let sub = toks.pop();
						let parent = toks.join("/");
						fdata.append("path", path);
					}
					r.open("post", "/api/get-entries");
					r.onerror = function() {
						printErrorStatus("network error occurred. please check whether the server is down.");
					}
					r.onload = function() {
						if (r.status != 200) {
							printErrorStatus("cannot get entry: " + req.responseText);
							return;
						}
						let resp = JSON.parse(r.responseText);
						let ents = resp.Msg;
						for (let ent of ents) {
							let path = ent.Path;
							if (!path.endsWith(sub)) {
								console.log("wanted to create sub entry" + sub + " but actaully created:" + path);
								return;
							}
							let parent = path.slice(0, path.length - sub.length - 1);
							if (parent == "") {
								parent = "/";
							}
							let prop = ent.Property;
							let tmpl = document.createElement("template");
							tmpl.innerHTML = `<div class="summaryDot summaryLabeler statusSelector grandSubEntry"></div>`;
							let gsub = tmpl.content.firstChild;
							gsub.innerText = sub;
							gsub.dataset.sub = sub;
							gsub.dataset.entryType = ent.Type;
							gsub.dataset.value = "";
							if (Object.hasOwnProperty(prop, "status")) {
								gsub.dataset.value = prop.status;
							}
							gsub.dataset.assignee = "";
							if (Object.hasOwnProperty(prop, "assignee")) {
								gsub.dataset.assignee = prop.status;
							}
							gsub.dataset.due = "";
							if (Object.hasOwnProperty(prop, "due")) {
								gsub.dataset.due = prop.due;
							}
							// temporary border for letting user notice new gsub entries. (until reload page)
							gsub.style.border = "1px solid #f84";
							let subEnt = document.querySelector(`.subEntry[data-entry-path="${parent}"]`);
							if (!subEnt) {
								// this could happen when user created non-direct child. eg. fx/main
								// TODO: handle this gracefully
								continue;
							}
							let gsubEnts = subEnt.querySelector(`.grandSubEntries`);
							gsubEnts.append(gsub);
							// TODO: hover on the new gsub entries doesn't work
						}
						let gsubArea = thisEnt.querySelector(".grandSubArea");
						gsubArea.classList.remove("adding");
						let adderInput = thisEnt.querySelector(".grandSubAdderInput");
						adderInput.innerHTML = "";
						printStatus("done");
					}
					r.send(fdata);
				}
				req.send(formData);
			}
		}
	}
	let infoTitles = document.getElementsByClassName("infoTitle");
	for (let t of infoTitles) {
		t.onclick = function(event) {
			if (subEntArea.contains(t) && !subEntArea.classList.contains("editMode")) {
				let info = t.closest(".subEntryInfo");
				let val = info.querySelector(".infoValue");
				if (val.classList.contains("expand")) {
					val.classList.remove("expand");
				} else {
					val.classList.add("expand")
				}
				return;
			}
			let info = t.closest(".info");
			let ent = info.closest(".entry");
			if (info.dataset.entryPath != ent.dataset.entryPath) {
				showInfoAdder(ent.dataset.entryPath, info.dataset.category, info.dataset.name, info.dataset.type, info.dataset.value);
				return;
			}
			showInfoUpdater(info);
		}
	}
	let infoSelectors = document.getElementsByClassName("infoSelector");
	for (let s of infoSelectors) {
		let tgl = s.closest(".infoCategoryToggle");
		s.onclick = function() {
			showCategoryInfos(tgl.dataset.category);
		}
	}
	let infoAdders = document.getElementsByClassName("infoAdder");
	for (let a of infoAdders) {
		let ent = a.closest(".entry");
		let tgl = a.closest(".infoCategoryToggle");
		a.onclick = function() {
			showInfoAdder(ent.dataset.entryPath, tgl.dataset.category, "", "text", "");
		}
	}
	let infoContextMenuLoaders = document.getElementsByClassName("infoContextMenuLoader");
	for (let loader of infoContextMenuLoaders) {
		loader.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let ent = loader.closest(".entry");
			let info = loader.closest(".info");
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
			infoDelete.classList.remove("nodisplay");
			if (info.dataset.category == "property") {
				infoDelete.classList.add("nodisplay");
			}
			if (info.dataset.category == "environ" && info.dataset.entryPath != ent.dataset.entryPath) {
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
	let pickedPropertyInput = document.querySelector(".propertyPickerValue");
	let resize = new ResizeObserver(function (entries) {
		pickedPropertyInput.dataset.resized = 1;
	});
	resize.observe(pickedPropertyInput);
	let dueLabels = document.querySelectorAll(".dueLabel")
	for (let el of dueLabels) {
		el.innerText = dday(el.dataset.due);
	}
	let assigneeLabels = document.querySelectorAll(".assigneeLabel")
	for (let el of assigneeLabels) {
		el.innerText = CalledByName[el.dataset.assignee];
	}
	let scrollToTop = document.querySelector("#scrollToTop");
	scrollToTop.onmouseenter = function() {
		scrollToTop.classList.add("reveal");
		scrollToTop.style.opacity = 1;
	}
	scrollToTop.onmouseleave = function() {
		scrollToTop.classList.remove("reveal");
		scrollToTop.style.opacity = scrollToTopOpacity();
	}
	window.onscroll = function(event) {
		if (scrollToTop.classList.contains("reveal")) {
			scrollToTop.style.opacity = 1;
		} else {
			scrollToTop.style.opacity = scrollToTopOpacity();
		}
	}
	scrollToTop.style.opacity = scrollToTopOpacity();
	let assetLinks = document.querySelectorAll(".assetLink");
	for (let link of assetLinks) {
		getEntry(
			link.dataset.entryPath,
			function(ent) {
				let dot = link.querySelector(".assetStatus");
				dot.dataset.entryType = ent.Type;
				let status = ent.Property["status"];
				if (status) {
					dot.dataset.value = status.Value;
				}
			},
			function(err) {
				link.classList.add("fail");
				link.title = err;
			},
		);
	}
	let keyshotLinks = document.querySelectorAll(".keyshotLink");
	for (let link of keyshotLinks) {
		getEntry(
			link.dataset.entryPath,
			function(ent) {
				let dot = link.querySelector(".keyshotStatus");
				dot.dataset.entryType = ent.Type;
				let status = ent.Property["status"];
				if (status) {
					dot.dataset.value = status.Value;
				}
			},
			function(err) {
				link.classList.add("fail");
				link.title = err;
			},
		);
	}
}

let EntryCache = {}

function getCachedEntry(path, onget, attempt) {
	if (attempt >= 5) {
		return;
	}
	let ent = EntryCache[path];
	if (ent) {
		onget(ent);
		return;
	}
	setTimeout(function() { getCachedEntry(path, onget, attempt+1) }, 200);
}

function getEntry(path, onget, onfail) {
	let ent = EntryCache[path];
	if (ent === null) {
		// checking the entry, but not quite complete yet.
		setTimeout(function() { getCachedEntry(path, onget, 0) }, 200);
	}
	if (ent) {
		onget(ent);
		return;
	}
	EntryCache[path] = null; // mark as the entry is on checking.
	let r = new XMLHttpRequest();
	let fdata = new FormData();
	fdata.append("path", path);
	r.open("post", "/api/get-entry");
	r.send(fdata);
	r.onerror = function() {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	r.onload = function() {
		let j = null;
		try {
			j = JSON.parse(r.responseText);
		} catch(err) {
			onfail(r.responseText);
			return;
		}
		if (j.Err != "") {
			onfail(j.Err);
			return;
		}
		let ent = j.Msg;
		EntryCache[path] = ent;
		onget(ent);
	}
}

function scrollToTopOpacity() {
	let op = (window.scrollY - 300) / 1000;
	op = Math.max(Math.min(op, 1), 0);
	return op * 0.1;
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

function submitForm(form) {
	let req = new XMLHttpRequest();
	req.open(form.method, form.action);
	req.send(new FormData(form));
	req.onload = function() {
		if (req.status == 200) {
			location.reload();
		} else {
			try {
				let j = JSON.parse(req.responseText);
				printErrorStatus(j.Err);
			} catch {
				printErrorStatus(req.responseText);
			}
		}
	}
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function goSearch(query) {
	let p = new URLSearchParams(window.location.search);
	if (query.startsWith("& ")) {
		// add to the current query
		let q = p.get("search");
		if (!q) {
			q = "";
		}
		let toks = query.split(" ").slice(1);
		for (let t of toks) {
			t = t.trim();
			if (!t) {
				continue;
			}
			if (q) {
				q += " ";
			}
			q += t;
		}
		query = q;
	} else {
		// user might have changed search_entry_type.
		// in '&' search case search type should not changed.
		let sel = document.querySelector("#searchTypeSelect");
		p.set("search_entry_type", sel.value)
	}
	p.set("search", query);
	let area = document.querySelector("#searchArea");
	location.href = area.dataset.searchFrom + "?" + p.toString();
}

function dday(due) {
	if (!due) {
		return "";
	}
	let then = new Date(due);
	let now = new Date();
	let today = new Date(now.toDateString());
	let day = 24 * 60 * 60 * 1000;
	let n = Math.floor((today - then) / day);
	let t = String(Math.abs(n))
	let c = "-"
	if (n > 0) {
		c = "+"
	}
	return "D" + c + t;
}

function removeClass(parent, clsName) {
	let elems = parent.getElementsByClassName(clsName);
	for (let e of elems) {
		e.classList.remove(clsName);
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

function updateQuickSearch(path, at, override) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("update_quick_search", "1");
	formData.append("quick_search_name", path);
	formData.append("quick_search_at", at);
	if (override) {
		formData.append("quick_search_override", "1");
	}
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
			let entryPath = thumb.closest(".entry").dataset.entryPath;
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
						refreshInfoValue(path, ctg, name, j.Msg);
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

function refreshInfoValue(path, ctg, name, p) {
	// If you are going to modify this function,
	// You should also modify 'infoValueElement' handler function in page_handler.go.

	let infoElem = document.querySelector(`.info[data-entry-path='${path}'][data-category='${ctg}'][data-name='${name}']`);
	if (!infoElem) {
		return
	}
	let valueElem = infoElem.querySelector(".infoValue");
	valueElem.innerHTML = "";
	let value = p.Value;
	infoElem.dataset.value = value;
	let evaled = p.Eval;
	if (p.Type == "tag") {
		let show = "";
		let toks = path.split("/");
		if (toks.length != 1) {
			show = toks[1];
		}
		for (let line of evaled.split("\n")) {
			line = line.trim();
			let a = document.createElement("a");
			a.classList.add("tagLink");
			a.href = "/"+show+"?search="+p.Name+"="+encodeURIComponent(line)
			let text = document.createTextNode(line);
			a.appendChild(text);
			valueElem.appendChild(a);
		}
	} else if (p.Type == "entry_link") {
		for (let line of evaled.split("\n")) {
			let pth = line.trim();
			if (pth == "") {
				continue;
			}
			let a = document.createElement("a");
			a.classList.add("entryLink");
			a.href = pth
			let text = document.createTextNode(pth);
			a.appendChild(text);
			valueElem.appendChild(a);
		}
	} else if (p.Type == "search") {
		for (let line of evaled.split("\n")) {
			line = line.trim();
			let toks = line.split(/[|](.*)/s);
			let name = toks[0];
			let query = toks[1];
			let div = document.createElement("div");
			div.classList.add("searchLink");
			div.dataset.searchFrom = p.Path;
			div.dataset.searchQuery = query;
			div.innerText = name;
			valueElem.appendChild(div);
		}
	} else {
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
			} else if (line.startsWith("https://")) {
				div.innerText = "";
				div.classList.add("externalLinkContainer");
				let icon = document.createElement("div");
				icon.classList.add("externalLinkIcon");
				div.appendChild(icon);
				let a = document.createElement("a");
				a.classList.add("externalLink");
				a.href = line;
				a.target = "_blank"; // open a new tab
				a.innerText = line.slice(8);
				div.appendChild(a);
			}
			valueElem.appendChild(div);
		}
	}
	// remove possible 'invalid' class
	valueElem.classList.remove("invalid");

	// Look UpdatedAt to check it was actually updated.
	// It might not, if new value is same as the old one.
	let updated = new Date(p.UpdatedAt);
	let now = Date.now();
	let delta = (now - updated);
	let day = 24 * 60 * 60 * 1000;
	if (delta <= day) {
		let dot = infoElem.querySelector(".recentlyUpdatedDot");
		let ent = dot.closest(".entry");
		let entDot = ent.querySelector(".recentlyUpdatedDot.forEntry");
		for (let d of [dot, entDot]) {
			d.dataset.updatedAt = p.UpdatedAt;
			d.title = "updated just now";
			d.classList.remove("invisible");
		}
	}
}

function showCategoryInfos(ctg) {
	let cont = document.querySelector(".mainEntryBottom");
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
	let thisEnt = info.closest(".entry");
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

function showInfoAdder(entry, ctg, name, type, value) {
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
			if (t == type) {
				option.selected = true;
			}
			option.value = t;
			option.text = t;
			typeSel.appendChild(option)
		}
	}
	adder.getElementsByClassName("valueForm")[0].action = "/api/add-" + ctg;

	let valueInput = adder.getElementsByClassName("valueInput")[0];
	valueInput.value = value;
	resizeTextArea(valueInput);
	if (!name) {
		nameInput.focus();
	} else {
		valueInput.focus();
	}
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

function printSelectionStatus() {
	let sel = document.querySelectorAll(".subEntry.selected");
	let what = "";
	let n = sel.length;
	if (n == 0) {
		what = "no entry";
	} else if (n == 1) {
		what = "1 entry";
	} else {
		what = String(n) + " entries";
	}
	printStatus(what + " selected");
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

function openDeleteEntryDialog(paths) {
	if (paths.length == 0) {
		printErrorStatus("no entry paths to delete");
		return;
	}
	let bg = document.querySelector("#deleteEntryDialogBackground");
	let dlg = document.querySelector("#deleteEntryDialog");
	let proms = [];
	for (let path of paths) {
		let p = new Promise((resolve, reject) => {
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("path", path);
			req.open("post", "/api/count-all-sub-entries");
			req.send(formData);
			req.onerror = function(err) {
				reject("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status != 200) {
					reject(req.responseText);
					return;
				}
				let j = JSON.parse(req.responseText);
				if (j.Err != "") {
					reject(j.Err);
					return;
				}
				let numSubEntries = j.Msg;
				resolve(path + " (+" + String(numSubEntries) + ")");
			}
		});
		proms.push(p);
	}
	Promise.all(proms).then(function(values) {
		let ents = "1 selected entry";
		if (paths.length > 1) {
			ents = String(paths.length) + " selected entries";
		}
		let content = "Do you really want to delete " + ents + "?\nIt will also delete their sub entries.\n";
		for (let v of values) {
			content += "\n" + v;
		}
		dlg.querySelector(".content").innerText = content;
		bg.classList.remove("invisible");
	}).catch(function(err) {
		printErrorStatus(err);
		return;
	})
	let cancelBtn = dlg.querySelector(".cancelButton");
	cancelBtn.onclick = function(ev) {
		ev.stopPropagation();
		bg.classList.add("invisible");
	}
	let confirmBtn = dlg.querySelector(".confirmButton");
	confirmBtn.onclick = function() {
		let req = new XMLHttpRequest();
		let formData = new FormData();
		for (let path of paths) {
			formData.append("path", path);
		}
		formData.append("recursive", "1");
		req.open("post", "/api/delete-entry");
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

let EnabledUserNames = [
{{- range $u := $.Users -}}
	"{{$u.Name}}",
{{end}}
];

let EnabledUserLabels = [
{{- range $u := $.Users -}}
	"{{$u.Called}} ({{$u.Name}})",
{{end}}
];

let AllUserNames = [
{{- range $u := $.Users -}}
	"{{$u.Name}}",
{{end}}
{{- range $u := $.DisabledUsers -}}
	"{{$u.Name}}",
{{end}}
];

let AllUserLabels = [
{{- range $u := $.Users -}}
	"{{$u.Called}} ({{$u.Name}})",
{{end}}
{{- range $u := $.DisabledUsers -}}
	"{{$u.Name}}",
{{end}}
];

// pun intended
let CalledByName = {
	"": "",
{{- range $u := $.Users -}}
	"{{$u.Name}}": "{{$u.Called}}",
{{end}}
{{- range $u := $.DisabledUsers -}}
	"{{$u.Name}}": "{{$u.Called}}",
{{end}}
}

let PossibleStatus = {
{{range $entType, $status := $.PossibleStatus}}
	"{{$entType}}": [
		{{with $status}}
		"", {{range $s := $status}}"{{$s.Name}}",{{end}}
		{{end}}
	],
{{end}}
}

let Properties = {
{{range $entType, $props := $.PropertyFilters}}
{{$hidden := index $.HiddenProperties $entType}}
	"{{$entType}}": [
		"",
		{{range $p :=  $props}}{{$p}},{{end}}
		{{range $p :=  $hidden}}{{$p}},{{end}}
	],
{{end}}
}

let LastPickedProperty = {
	{{range $entType, $p := $.UserSetting.PickedProperty}}
	"{{$entType}}": "{{$p}}",
	{{end}}
}

// autoComplete takes input tag and possible autocompleted values and label.
// It takes oncomplete function as an argument that will be called with user selected value.
// It will give oncomplete raw input value when it cannot find any item with the value.
// it returns clean function which unbind handlers for autoComplete of the input.
function autoComplete(input, labels, vals, menuAt, oncomplete) {
	// Turn off browser's default autocomplete behavior.
	input.setAttribute("autocomplete", "off");
	let focus = -1;
	let oninput = function(event) {
		let search = input.value;
		if (search == "") {
			return;
		}
		let lsearch = search.toLowerCase();
		// reset focus on further input.
		focus = -1;
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
	let onkeydown = function(event) {
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
			if (input.value == "") {
				oncomplete("");
				return;
			}
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
	}
	let onkeyup = function(event) {
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
	input.addEventListener("input", oninput);
	input.addEventListener("keydown", onkeydown);
	input.addEventListener("keyup", onkeyup);
	// clean clears event handlers binded in this function.
	function clean() {
		input.removeEventListener("input", oninput);
		input.removeEventListener("keydown", onkeydown);
		input.removeEventListener("keyup", onkeyup);
	}
	return clean;
}

// cleanAutoComplete is a function clears autoComplete handlers binded to .propertyPickerValue.
// Feel not so good, but I couldn't think better way to clear it. At least for now.
let cleanAutoComplete = null;

function reloadPropertyPicker(popup, prop) {
	LastPickedProperty[popup.dataset.entryType] = prop;
	let nameInput = popup.querySelector(".propertyPickerName");
	let valueInput = popup.querySelector(".propertyPickerValue");
	let history = popup.querySelector(".propertyPickerHistory");
	nameInput.dataset.value = prop;
	if (prop == "") {
		nameInput.dataset.error = "";
		nameInput.dataset.modified = "";
		valueInput.disabled = "1";
		valueInput.value = "";
		history.classList.add("hidden");
		return;
	}
	valueInput.disabled = "";
	history.classList.remove("hidden");
	let mainDiv = document.querySelector(".main");
	let entPath = popup.dataset.entryPath;
	let path = entPath;
	if (popup.dataset.sub != "") {
		path += "/" + popup.dataset.sub
	}
	let updateInputs = function(type, value) {
		if (cleanAutoComplete != null) {
			cleanAutoComplete();
			cleanAutoComplete = null;
		}
		nameInput.dataset.type = type;
		nameInput.dataset.error = "";
		nameInput.dataset.modified = "";
		valueInput.value = value;
	}
	if (prop == "*environ") {
		getEntryEnvirons(path, function(envs) {
			let environs = [];
			for (let e of envs) {
				let l = e.Name + "=" + e.Value;
				if (e.Path != path) {
					l = "~" + l;
				}
				environs.push(l);
			}
			environs.sort();
			updateInputs("environ", environs.join("\n"));
			printStatus("done");
		});
	} else if (prop == "*access") {
		getEntryAccessList(path, function(accs) {
			let accessList = [];
			for (let a of accs) {
				let l = a.Name + "=" + a.Value;
				if (a.Path != path) {
					l = "~" + l;
				}
				accessList.push(l);
			}
			accessList.sort();
			updateInputs("access", accessList.join("\n"));
			printStatus("done");
		});
		return;
	} else {
		getProperty(path, prop, function(p) {
			updateInputs(p.Type, p.Eval);
			if (nameInput.dataset.type == "user") {
				let menuAt = getOffset(valueInput);
				menuAt.top += valueInput.getBoundingClientRect().height + 4;
				cleanAutoComplete = autoComplete(valueInput, AllUserLabels, AllUserNames, menuAt, function(value) {
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
					let sub = popup.dataset.sub;
					let paths = []
					for (let ent of selectedEnts) {
						let path = ent.dataset.entryPath;
						if (sub != "") {
							if (ent.querySelector(`.grandSubEntry[data-sub="${sub}"]`) == null) {
								continue
							}
							path += "/" + sub;
						}
						paths.push(path);
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
					updateProperty(paths, nameInput.value, value, onsuccess);
				});
			}
			printStatus("done");
		});
	}
}

function getProperty(path, prop, onsuccess) {
	let data = new FormData();
	data.append("path", path)
	data.append("name", prop)
	postForge("/api/get-property", data, onsuccess);
}

function getProperties(paths, prop, onsuccess) {
	let data = new FormData();
	for (let path of paths) {
		data.append("path", path)
		data.append("name", prop)
	}
	postForge("/api/get-properties", data, onsuccess);
}

function getEntryEnvirons(path, onsuccess) {
	let data = new FormData();
	data.append("path", path)
	postForge("/api/entry-environs", data, onsuccess);
}

function getEntryAccessList(path, onsuccess) {
	let data = new FormData();
	data.append("path", path)
	postForge("/api/entry-access-list", data, onsuccess);
}

function updateProperty(paths, prop, value, onsuccess) {
	let data = new FormData();
	for (let path of paths) {
		data.append("path", path);
	}
	data.append("name", prop);
	data.append("value", value);
	postForge("/api/update-property", data, onsuccess);
}

function postForge(api, data, onsuccess) {
	let r = new XMLHttpRequest();
	r.open("post", api);
	r.send(data);
	r.onerror = function() {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	r.onload = function() {
		if (r.status != 200) {
			printErrorStatus(r.responseText);
			return;
		}
		// update api doesn't respond anything, when it is done without an error.
		if (!r.responseText) {
			onsuccess();
			return;
		}
		let j = JSON.parse(r.responseText);
		if (j.Err != "") {
			printErrorStatus(j.Err);
			return;
		}
		onsuccess(j.Msg);
	}
}
