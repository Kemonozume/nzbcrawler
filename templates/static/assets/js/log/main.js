var app = LogObj


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
function AddTag() {
    app.addTag(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

function AddLevel() {
    app.addLevel(document.getElementById('abtags_input').value)
    document.getElementById('abtags_input').value = ""
}

