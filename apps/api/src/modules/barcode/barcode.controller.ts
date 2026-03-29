import { Body, Controller, Post } from "@nestjs/common";
import type { AuthUser, BarcodeLookupResponse } from "@media-library/types";
import { CurrentUser } from "../auth/decorators/current-user.decorator";
import { BarcodeLookupDto } from "./dto/barcode-lookup.dto";
import { BarcodeService } from "./barcode.service";

@Controller("barcode")
export class BarcodeController {
  constructor(private readonly barcodeService: BarcodeService) {}

  @Post("lookup")
  lookup(
    @CurrentUser() user: AuthUser,
    @Body() body: BarcodeLookupDto
  ): Promise<BarcodeLookupResponse> {
    return this.barcodeService.lookup(user.id, body);
  }
}
