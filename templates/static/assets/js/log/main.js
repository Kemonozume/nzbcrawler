var app = LogObj
var filter = ""


app.activateScrollBinding()
app.buildUI()
app.addLogs()


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

function scrollToTop() {
    document.getElementById("releases").scrollTop = 0
}

function Filter() {
    var filter = document.getElementById("filter").value
    $("#cont tr").each(function(index) {
        text = $(this).children().last()[0].innerText
        console.log(text)
        console.log(filter)
        if (text.indexOf(filter) != -1) {
            $(this).show()
        }else {
            $(this).hide()
        }
    })
}

function AddTag() {
    app.addTag(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

function AddLevel() {
    app.addLevel(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

