// loaded for checkout and login page

// Display the backend error on the page
function displayErr(type) {
    if (type === "combo_fail") {
        return "Wrong email / password combination";
    } else if (type === "stripe_cancel") {
        return "Error during payment process";
    } else if (type === "failed_crypt" || type === "failed_session") {
        return "internal error [" + type + "] Please contact administrator (info@theotowngarage.com)";
    } else if (type === "wrong_email") {
        return "Your email is not registered with us";
    } else if (type === "token_invalid") {
        return "Reset token expired or invalid";
    } else {
        return "Sorry, an error happened [" + type + "] Please contact administrator (info@theotowngarage.com)";
    }
}

const paramsString = window.location.search;
const searchParams = new URLSearchParams(paramsString);
if(searchParams.has("reason")) {
    err_msg = document.getElementById("err_message");
    err_msg.innerHTML = displayErr(searchParams.get("reason"));
    document.getElementById("error").classList.remove("invisible");
}
