var patt = new RegExp("^(.*?)(German|german|1080|720|[0-9]{4})");
var app = AppObj

$.getJSON("/db/tags/", function(data2) {
    if ($.isEmptyObject(data2)) {
        return
    }
    $("#cloudlist").append(_.template($('#tags-list-template').html(), {tags: data2}));
})


app.activateScrollBinding()
app.buildUI()
app.addReleases()

function getNZB(view, id) {
    $.get("/db/event/"+id+"/thank", function( data ) {
      var li = view.parentNode
      $(li).empty()
      $(li).append("<a target='_blank' href='/db/event/"+data.id+"/nzb'>nzb download</a>")
      $(li.parentNode.parentNode.parentNode.childNodes[3]).append("<br><br><span class='password'>Password: "+data.passwort+"</span>")
    });
}

function toggleActionbar() {
    if (document.getElementById("toggleaction").innerText == "▼") {
        
        $("#releases").animate({
            height: "85%" 
        }, 300, function() {
            $("#ab").show("fast")
        })
        document.getElementById("toggleaction").innerText = "▲"
    }else {
        $("#ab").hide("fast", function() {
            $("#releases").animate({
                height: "90%" 
            }, 300)
        })
        document.getElementById("toggleaction").innerText = "▼"
        
    }
}

function toggleCloud() {
    if($("#cloud").is(":visible")) {
        $("#cloud").hide()
        $("#releases").show()
    }else {
        $("#cloud").show()
        $("#releases").hide()
    }
}

function AddTag() {
    app.addTag(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

function AddName() {
    app.addName(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

function scrollToTop() {
    document.getElementById("releases").scrollTop = 0
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

String.prototype.pad = function(l, s, t){
    return s || (s = " "), (l -= this.length) > 0 ? (s = new Array(Math.ceil(l / s.length)
        + 1).join(s)).substr(0, t = !t ? l : t == 1 ? 0 : Math.ceil(l / 2))
        + this + s.substr(0, l - t) : this;
};
