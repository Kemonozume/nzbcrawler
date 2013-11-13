var patt = new RegExp("^(.*?)(German|german|1080|720|[0-9]{4})");
var offset = 0;
var tags = "";
var name = "";
var loading = false;
document.onmousewheel = this.checkScroll;
addUI();


$("#tags").hide();
$("#showtag").click(function () {
    if ($("#tags")[0].style.display == "none") {
        $("#tags").show();
    } else {
        $("#tags").hide();
    }

});

$("#text-tags").keyup(function () {
    if (tags === $("#text-tags").val()) {
        return
    }
    tags = $("#text-tags").val()
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
});
$("#text-name").keyup(function () {
    if (name === $("#text-name").val()) {
        return
    }
    name = $("#text-name").val()
    $("#tcont").html("");
    document.getElementById('tcont').style.top = 0;
    offset = 0;
    addUI();
});

$("#movpicker").click(function () {
    $("#text-tags").val("movies&hd");
    tags = $("#text-tags").val()
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
});
$("#cinpicker").click(function () {
    $("#text-tags").val("cinema&hd");
    tags = $("#text-tags").val()
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
});
$("#pcpicker").click(function () {
    $("#text-tags").val("pc");
    tags = $("#text-tags").val()
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
});
$("#serienpicker").click(function () {
    $("#text-tags").val("serie&hd");
    tags = $("#text-tags").val()
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
});



function upvote(str, obj) {
    $.get("/db/event/" + str + "/score/1", function (data) {
        var obj2 = $(obj.parentNode).find("p")[0];
        var rating = parseInt(obj2.innerHTML.split(":")[1]);
        if (rating == -1) {
            rating == 1;
        } else {
            rating++;
        }
        obj2.innerHTML = "Rating: " + rating;
    });
}

function downvote(str, obj) {
    $.get("/db/event/" + str + "/score/-1", function (data) {
        var obj2 = $(obj.parentNode).find("p")[0];
        var rating = parseInt(obj2.innerHTML.split(":")[1]);
        if (rating == 1) {
            rating == -1;
        } else {
            rating--;
        }
        obj2.innerHTML = "Rating: " + rating;
    });
}

function showoverlay(bla) {
    var obj = $(bla).find(".overlay")[0]
    if (obj.style.display == "none") {
        obj.style.display = "block";
    } else {
        obj.style.display = "none";
    }
}

function getMovieName(str) {
    var bla = patt.exec(str);
    if (bla != undefined || bla != null) {
        var name = bla[1];
        if (name != "" || name != null) {
            name = name.trim();
            name = name.replace(/\./g, " ");
            return name;
        } else {
            return null;
        }
    } else {
        return null;
    }
}

function checkValid(str) {
    astr = str.split("")
    klammer = 0
    valid = false
    for (i = 0; i < astr.length; i++) {
        if (astr[i] === "(") {
            klammer++
        }
        if (astr[i] === ")") {
            klammer--
        }
        if (/[a-zA-Z0-9]/.test(astr[i])) {
            if (/[a-zA-Z0-9]/.test(astr[i + 1]) || astr[i] === "(" || astr[i + 1] === ")" || i + 1 == astr.length || astr[i + 1] === "|" || astr[i + 1] === "&") {} else {
                break
            }
        }
        if (astr[i] === "&") {
            if ((/[a-zA-Z0-9]/.test(astr[i + 1]) || astr[i + 1] === "(" || astr[i + 1] === "!") && i + 1 != astr.length) {} else {
                break
            }
        }
        if (astr[i] === "|") {
            if ((/[a-zA-Z0-9]/.test(astr[i + 1]) || astr[i + 1] === "(" || astr[i + 1] === "!") && i + 1 != astr.length) {} else {
                break
            }
        }
        if (astr[i] === "!") {
            if ((/[a-zA-Z0-9]/.test(astr[i + 1])) && i + 1 != astr.length) {} else {
                break
            }
        }
        if (i == astr.length - 1) {
            if (klammer == 0) {
                valid = true
            }
        }

    }
    return valid
}

function getCmd() {
    var cmd = "";
    if (tags == "") {
        cmd += "none"
    } else {
        tags = tags.replace(/\//g, "");
        cmd += tags;
    }
    if (name == "") {
        cmd += "/none"
    } else {
        cmd += ("/" + escape(name))
    }
    return cmd;
}

function setTag(str) {
    $("#text-tags").val(str);
    tags = $("#text-tags").val();
    $("#tags").hide();
    if (checkValid(tags)) {
        $("#tcont").html("");
        document.getElementById('tcont').style.top = 0;
        offset = 0;
        addUI();
    }
}

function checkScroll(event) {
    var delta = 0;

    if (!event) event = window.event;

    if (event.wheelDelta) {
        delta = event.wheelDelta / 60;
    } else if (event.detail) {
        delta = -event.detail / 2;
    }

    var currPos = document.getElementById('tcont').offsetTop;
    var amount2 = currPos * -1;

    if (amount2 > $("#tcont").outerHeight() - 1800 && !loading) {
        offset += 200;
        addUI();
    }

    if (delta < 0) currPos = parseInt(currPos) - (128 + 10);
    else currPos = parseInt(currPos) + (128 + 10);
    if (currPos > 0) currPos = 0;

    document.getElementById('tcont').style.top = currPos + "px";
}

function addUI() {
    if (loading == true) return;
    loading = true;
    $.getJSON('/db/events/' + offset + "/" + getCmd(), function (data) {
        if ($.isEmptyObject(data)) {
            loading = false;
            return;
        }
        $("#tcont").append(_.template($('#release-list-template').html(), {releases: data}));
        loading = false;
        offset+=200;
    });
}
