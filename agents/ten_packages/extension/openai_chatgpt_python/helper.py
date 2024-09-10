#
#
# Agora Real Time Engagement
# Created by Wei Hu in 2024-08.
# Copyright (c) 2024 Agora IO. All rights reserved.
#
#
from ten.data import Data
from .log import logger
from PIL import Image
from datetime import datetime
from io import BytesIO
from base64 import b64encode


def get_property_bool(data: Data, property_name: str) -> bool:
    """Helper to get boolean property from data with error handling."""
    try:
        return data.get_property_bool(property_name)
    except Exception as err:
        logger.warn(f"GetProperty {property_name} failed: {err}")
        return False

def get_property_string(data: Data, property_name: str) -> str:
    """Helper to get string property from data with error handling."""
    try:
        return data.get_property_string(property_name)
    except Exception as err:
        logger.warn(f"GetProperty {property_name} failed: {err}")
        return ""
    
def get_property_int(data: Data, property_name: str) -> int:
    """Helper to get int property from data with error handling."""
    try:
        return data.get_property_int(property_name)
    except Exception as err:
        logger.warn(f"GetProperty {property_name} failed: {err}")
        return 0
    
def get_property_float(data: Data, property_name: str) -> float:
    """Helper to get float property from data with error handling."""
    try:
        return data.get_property_float(property_name)
    except Exception as err:
        logger.warn(f"GetProperty {property_name} failed: {err}")
        return 0.0
    

def get_current_time():
    # Get the current time
    start_time = datetime.now()
    # Get the number of microseconds since the Unix epoch
    unix_microseconds = int(start_time.timestamp() * 1_000_000)
    return unix_microseconds


def is_punctuation(char):
    if char in [",", "，", ".", "。", "?", "？", "!", "！"]:
        return True
    return False


def parse_sentence(sentence, content):
    remain = ""
    found_punc = False

    for char in content:
        if not found_punc:
            sentence += char
        else:
            remain += char

        if not found_punc and is_punctuation(char):
            found_punc = True

    return sentence, remain, found_punc


def rgb2base64jpeg(rgb_data, width, height):
    # Convert the RGB image to a PIL Image
    pil_image = Image.frombytes("RGBA", (width, height), bytes(rgb_data))
    pil_image = pil_image.convert("RGB")

    # Resize the image while maintaining its aspect ratio
    pil_image = resize_image_keep_aspect(pil_image, 320)

    # Save the image to a BytesIO object in JPEG format
    buffered = BytesIO()
    pil_image.save(buffered, format="JPEG")
    # pil_image.save("test.jpg", format="JPEG")

    # Get the byte data of the JPEG image
    jpeg_image_data = buffered.getvalue()

    # Convert the JPEG byte data to a Base64 encoded string
    base64_encoded_image = b64encode(jpeg_image_data).decode("utf-8")

    # Create the data URL
    mime_type = "image/jpeg"
    base64_url = f"data:{mime_type};base64,{base64_encoded_image}"
    return base64_url


def resize_image_keep_aspect(image, max_size=512):
    """
    Resize an image while maintaining its aspect ratio, ensuring the larger dimension is max_size.
    If both dimensions are smaller than max_size, the image is not resized.

    :param image: A PIL Image object
    :param max_size: The maximum size for the larger dimension (width or height)
    :return: A PIL Image object (resized or original)
    """
    # Get current width and height
    width, height = image.size

    # If both dimensions are already smaller than max_size, return the original image
    if width <= max_size and height <= max_size:
        return image

    # Calculate the aspect ratio
    aspect_ratio = width / height

    # Determine the new dimensions
    if width > height:
        new_width = max_size
        new_height = int(max_size / aspect_ratio)
    else:
        new_height = max_size
        new_width = int(max_size * aspect_ratio)

    # Resize the image with the new dimensions
    resized_image = image.resize((new_width, new_height))

    return resized_image