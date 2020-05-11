$(document).ready(function() {
    $("#button1").click(function(){
        $(".fadeout").fadeOut();
    });
    $(".otherbutton").click(function(){
        $(".fadeout").fadeIn();
    });
    $("#togglebutton").click(function(){
        $(".fadeout").toggle();
    });
});

