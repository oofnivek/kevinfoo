use actix_web::{get, post, web, HttpResponse, Responder};
use handlebars::Handlebars;
use serde::Deserialize;
use serde_json::json;
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine as _};
use hmac::{Hmac, Mac};
use sha2::Sha256;

#[derive(Deserialize)]
pub struct DecodeRequest {
    pub token: String,
}

#[derive(Deserialize)]
pub struct MintRequest {
    pub header: String,
    pub payload: String,
    pub secret: String,
}

#[get("/jwt")]
pub async fn jwt_index(hb: web::Data<Handlebars<'_>>) -> impl Responder {
    let body = hb.render("jwt", &json!({})).unwrap();
    let html = hb.render("base", &json!({
        "title": "JWT Decoder",
        "body": body
    })).unwrap();
    HttpResponse::Ok().body(html)
}

#[post("/jwt/decode")]
pub async fn decode_jwt(
    hb: web::Data<Handlebars<'_>>,
    form: web::Form<DecodeRequest>,
) -> impl Responder {
    let parts: Vec<&str> = form.token.split('.').collect();
    
    if parts.len() < 2 {
        let body = hb.render("jwt_result", &json!({
            "header": "",
            "payload": "",
            "error": "Invalid JWT format. Must have at least 2 segments."
        })).unwrap();
        return HttpResponse::Ok().body(body);
    }

    let decode = |s: &str| {
        URL_SAFE_NO_PAD.decode(s)
            .ok()
            .and_then(|b| String::from_utf8(b).ok())
            .and_then(|s| serde_json::from_str::<serde_json::Value>(&s).ok())
            .map(|v| serde_json::to_string_pretty(&v).unwrap_or_default())
            .unwrap_or_else(|| "{}".to_string())
    };

    let header = decode(parts[0]);
    let payload = decode(parts[1]);

    let body = hb.render("jwt_result", &json!({
        "header": header,
        "payload": payload
    })).unwrap();
    
    HttpResponse::Ok().body(body)
}

#[post("/jwt/mint")]
pub async fn mint_jwt(
    form: web::Form<MintRequest>,
) -> impl Responder {
    // 1. Verify JSON validity (optional but good for UX)
    if let Err(e) = serde_json::from_str::<serde_json::Value>(&form.header) {
        return HttpResponse::BadRequest().body(format!("Invalid Header JSON: {}", e));
    }
    if let Err(e) = serde_json::from_str::<serde_json::Value>(&form.payload) {
        return HttpResponse::BadRequest().body(format!("Invalid Payload JSON: {}", e));
    }

    // 2. Base64URL encode exactly what the user provided in the textareas
    // We remove newlines and extra spaces for a compact token, but preserve the field sequence
    // Actually, to be safe and "standard", we should probably minify it first.
    let minify = |s: &str| {
        serde_json::from_str::<serde_json::Value>(s)
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .unwrap_or_default()
    };

    let header_compact = minify(&form.header);
    let payload_compact = minify(&form.payload);

    let header_b64 = URL_SAFE_NO_PAD.encode(header_compact.as_bytes());
    let payload_b64 = URL_SAFE_NO_PAD.encode(payload_compact.as_bytes());

    let signing_input = format!("{}.{}", header_b64, payload_b64);

    // 3. Sign using HMAC-SHA256 (HS256)
    type HmacSha256 = Hmac<Sha256>;
    let mut mac = match HmacSha256::new_from_slice(form.secret.as_bytes()) {
        Ok(m) => m,
        Err(e) => return HttpResponse::InternalServerError().body(format!("Invalid key length: {}", e)),
    };
    mac.update(signing_input.as_bytes());
    let signature = mac.finalize().into_bytes();
    let signature_b64 = URL_SAFE_NO_PAD.encode(&signature);

    let token = format!("{}.{}", signing_input, signature_b64);

    HttpResponse::Ok().body(format!(
        r#"<div class="notification is-success is-light">
            <label class="label">Minted Token (Preserving Header/Payload content)</label>
            <textarea class="textarea is-family-monospace" readonly onclick="this.select()">{}</textarea>
           </div>"#, token
    ))
}
