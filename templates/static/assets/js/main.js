var patt = new RegExp("^(.*?)(German|german|1080|720|[0-9]{4})");

toastr.options = {
  "closeButton": false,
  "debug": false,
  "positionClass": "toast-bottom-right",
  "onclick": null,
  "showDuration": "300",
  "hideDuration": "1000",
  "timeOut": "3000",
  "extendedTimeOut": "1000",
  "showEasing": "linear",
  "hideEasing": "linear",
  "showMethod": "fadeIn",
  "hideMethod": "fadeOut"
}

String.prototype.pad = function(l, s, t){
    return s || (s = " "), (l -= this.length) > 0 ? (s = new Array(Math.ceil(l / s.length)
        + 1).join(s)).substr(0, t = !t ? l : t == 1 ? 0 : Math.ceil(l / 2))
        + this + s.substr(0, l - t) : this;
};

var route = RouteObj;
var app = AppObj;
var log = LogObj;

app.activateScrollBinding();
log.activateScrollBinding();

route.addRoute("/", function() {
    app.reset();
    app.addUI();
});
route.addRoute("/stats", function() {
    log.reset();
    log.addUI();
});
route.addRoute("/logs", function() {
    log.reset();
    log.addUI();
});
route.addRoute("/movies", function() {
    app.reset();
    app.genre = "movie";
    app.addUI();
});
route.addRoute("/movies/hd", function() {
    app.reset();
    app.genre = "movie&hd";
    app.addUI();
});
route.addRoute("/movies/hd", function() {
    app.reset();
    app.genre = "movie&hd";
    app.addUI();
});
route.addRoute("/movies/sd", function() {
    app.reset();
    app.genre = "movie&sd";
    app.addUI();
});
route.addRoute("/cinema", function() {
    app.reset();
    app.genre = "cinema";
    app.addUI();
});
route.addRoute("/cinema/hd", function() {
    app.reset();
    app.genre = "cinema&hd";
    app.addUI();
});
route.addRoute("/cinema/sd", function() {
    app.reset();
    app.genre = "cinema&sd";
    app.addUI();
});
route.addRoute("/serie", function() {
    app.reset();
    app.genre = "serie";
    app.addUI();
});
route.addRoute("/serie/hd", function() {
    app.reset();
    app.genre = "serie&hd";
    app.addUI();
});
route.addRoute("/serie/sd", function() {
    app.reset();
    app.genre = "serie&sd";
    app.addUI();
});
route.addRoute("/pc", function() {
    app.reset();
    app.genre = "pc";
    app.addUI();
});
route.create();


$('#s-button').bind("click", function() {
    parent.location.hash = "/search";
    app.reset();
    var bla = document.getElementById("s-name").value;
    var bla2 = document.getElementById("s-genre").value;
    if(bla == "") {
        bla = "none";
    }
    if(bla2 == "") {
        bla2 = "none";
    }

    app.genre = bla2;
    app.name = bla;
    app.addUI();
});


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
