import { Module } from "@nestjs/common";
import { LibraryModule } from "../library/library.module";
import { MediaModule } from "../media/media.module";
import { ProvidersModule } from "../providers/providers.module";
import { BarcodeController } from "./barcode.controller";
import { BarcodeService } from "./barcode.service";

@Module({
  imports: [ProvidersModule, MediaModule, LibraryModule],
  controllers: [BarcodeController],
  providers: [BarcodeService]
})
export class BarcodeModule {}
