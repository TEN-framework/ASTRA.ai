#
#
# Agora Real Time Engagement
# Created by XinHui Li in 2024-07.
# Copyright (c) 2024 Agora IO. All rights reserved.
#
#

from rte import (
    Addon,
    register_addon_as_extension,
    RteEnv,
)
from .log import logger


@register_addon_as_extension("interrupt_detector_python")
class InterruptDetectorExtensionAddon(Addon):
    def on_init(self, rte: RteEnv, manifest, property) -> None:
        logger.info("on_init")
        rte.on_init_done(manifest, property)
        return

    def on_create_instance(self, rte: RteEnv, addon_name: str, context) -> None:
        logger.info("on_create_instance")

        from .interrupt_detector_extension import InterruptDetectorExtension

        rte.on_create_instance_done(InterruptDetectorExtension(addon_name), context)

    def on_deinit(self, rte: RteEnv) -> None:
        logger.info("on_deinit")
        rte.on_deinit_done()
        return
