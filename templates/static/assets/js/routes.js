var RouteObj = {
	routes: {},

	addRoute: function(str, fun) {
		this.routes[str] = fun;
	},

	interpretUrl: function(url) {
		var bl = url.split("/");
	    var bl2 = [];
	    for(var i = 4; i < bl.length; i++) {
	        bl2.push("/"+bl[i]);
	    }
	    var name = "";
		if(bl2.length == 0) {
			parent.location.hash = "/";
		}else if (bl2.length == 1 || bl2[0] == "/search") {
			name = bl2[0];
		}else {
			name = bl2.join("");
		}
		$("#tcont").empty().hide();
		$("#lcont").empty().hide();

		this.routes[name]();
		$("nav li").removeClass("active");
		var li = $('a[href="#'+name+'"]')[0];
		if($(li).parent().parent().hasClass("dropdown-menu")) {
			$(li).parent().parent().parent().addClass("active");
		}else {
			$(li).parent().addClass("active");
		}		
	},

	hashhand: function(){
    	this.oldHash = window.location.hash;
    	this.Check;

    	var that = this;
    	var detect = function(){
        	if(that.oldHash!=window.location.hash){
            	that.oldHash = window.location.hash;
            	that.interpretUrl(document.URL);
        	}
    	};
    	this.Check = setInterval(function(){ detect() }, 100);
    },

	create: function() {
		this.hashhand();
		this.interpretUrl(document.URL);
	}
}